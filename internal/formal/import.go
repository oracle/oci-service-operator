package formal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

const defaultProviderSourceName = "terraform-provider-oci"
const defaultResponseItemsField = "Items"

type ImportOptions struct {
	Root             string
	ProviderPath     string
	ProviderRevision string
	Service          string
	SourceName       string
}

type ImportReport struct {
	Root                string
	ModulePath          string
	Revision            string
	SourceName          string
	ImportsRefreshed    int
	ScaffoldRowsSkipped int
}

func (r ImportReport) String() string {
	var b strings.Builder
	b.WriteString("formal import completed\n")
	fmt.Fprintf(&b, "- root: %s\n", filepath.ToSlash(r.Root))
	fmt.Fprintf(&b, "- source: %s\n", r.SourceName)
	fmt.Fprintf(&b, "- module path: %s\n", r.ModulePath)
	fmt.Fprintf(&b, "- revision: %s\n", r.Revision)
	fmt.Fprintf(&b, "- imports refreshed: %d\n", r.ImportsRefreshed)
	fmt.Fprintf(&b, "- scaffold rows skipped: %d\n", r.ScaffoldRowsSkipped)
	return b.String()
}

type providerImportAnalysis struct {
	ModulePath string
	Revision   string
	Resources  map[string]providerResourceFacts
	Skipped    map[string]string
}

type providerResourceFacts struct {
	Operations         operations
	Lifecycle          lifecycle
	Mutation           mutation
	Hooks              hooks
	DeleteConfirmation lifecyclePhase
	ListLookup         *listLookup
}

type providerPackageContext struct {
	PkgPath string
	Funcs   map[string]*providerFuncContext
	Methods map[string]map[string]*providerFuncContext
}

type providerFuncContext struct {
	Decl         *ast.FuncDecl
	Imports      map[string]string
	ReceiverName string
}

type providerRegistration struct {
	TerraformName string
	Constructor   string
	Package       *providerPackageContext
}

type listLookupFacts struct {
	Bindings []operationBinding
	Lookup   listLookup
}

type schemaField struct {
	Path          string
	Optional      bool
	Required      bool
	Computed      bool
	ForceNew      bool
	ConflictsWith []string
}

var nonRecursiveReceiverMethods = map[string]struct{}{
	"Create":         {},
	"Get":            {},
	"Update":         {},
	"Delete":         {},
	"SetData":        {},
	"VoidState":      {},
	"ID":             {},
	"State":          {},
	"CreatedPending": {},
	"CreatedTarget":  {},
	"UpdatedPending": {},
	"UpdatedTarget":  {},
	"DeletedPending": {},
	"DeletedTarget":  {},
}

func Import(opts ImportOptions) (ImportReport, error) {
	report := ImportReport{
		SourceName: strings.TrimSpace(opts.SourceName),
	}
	if report.SourceName == "" {
		report.SourceName = defaultProviderSourceName
	}

	if strings.TrimSpace(opts.Root) == "" {
		return report, fmt.Errorf("formal root must not be empty")
	}
	if strings.TrimSpace(opts.ProviderPath) == "" {
		return report, fmt.Errorf("provider source path must not be empty")
	}

	formalRoot, err := filepath.Abs(strings.TrimSpace(opts.Root))
	if err != nil {
		return report, err
	}
	report.Root = formalRoot
	if err := requireDirectory(formalRoot); err != nil {
		return report, err
	}

	providerRoot, err := resolveProviderRoot(strings.TrimSpace(opts.ProviderPath))
	if err != nil {
		return report, err
	}

	rows, err := loadManifest(filepath.Join(formalRoot, "controller_manifest.tsv"))
	if err != nil {
		return report, err
	}
	sourceLock, _, err := loadSourceLock(filepath.Join(formalRoot, "sources.lock"))
	if err != nil {
		return report, err
	}

	analysis, err := analyzeProviderSource(providerRoot, strings.TrimSpace(opts.ProviderRevision))
	if err != nil {
		return report, err
	}
	report.ModulePath = analysis.ModulePath
	report.Revision = analysis.Revision

	sourceLock.Sources = upsertSource(sourceLock.Sources, sourceLockEntry{
		Name:     report.SourceName,
		Surface:  "provider-facts",
		Status:   "pinned",
		Path:     analysis.ModulePath,
		Revision: analysis.Revision,
		Notes: []string{
			"Pinned module path and revision imported by formal-import.",
		},
	})

	if err := writeJSONFile(filepath.Join(formalRoot, "sources.lock"), sourceLock); err != nil {
		return report, err
	}

	serviceFilter := strings.TrimSpace(opts.Service)
	for _, row := range rows {
		if serviceFilter != "" && row.Service != serviceFilter {
			continue
		}
		if row.Stage == "scaffold" {
			report.ScaffoldRowsSkipped++
			continue
		}

		importPath := filepath.Join(formalRoot, filepath.Clean(row.ImportPath))
		currentDoc, err := loadImport(importPath)
		if err != nil {
			return report, fmt.Errorf("%s: %w", filepath.ToSlash(row.ImportPath), err)
		}
		if currentDoc.SourceRef != "" && currentDoc.SourceRef != report.SourceName {
			return report, fmt.Errorf("%s: sourceRef=%q does not match importer source %q", filepath.ToSlash(row.ImportPath), currentDoc.SourceRef, report.SourceName)
		}
		if strings.TrimSpace(currentDoc.ProviderResource) == "" {
			return report, fmt.Errorf("%s: providerResource must not be empty", filepath.ToSlash(row.ImportPath))
		}

		facts, ok := analysis.Resources[currentDoc.ProviderResource]
		if !ok {
			if reason, skipped := analysis.Skipped[currentDoc.ProviderResource]; skipped {
				return report, fmt.Errorf("%s: provider resource %q is registered but unsupported by formal-import: %s", filepath.ToSlash(row.ImportPath), currentDoc.ProviderResource, reason)
			}
			return report, fmt.Errorf("%s: provider resource %q not found in %s", filepath.ToSlash(row.ImportPath), currentDoc.ProviderResource, analysis.ModulePath)
		}

		updated := currentDoc
		updated.SchemaVersion = currentSchemaVersion
		updated.Surface = "provider-facts"
		updated.Service = row.Service
		updated.Slug = row.Slug
		updated.Kind = row.Kind
		updated.SourceRef = report.SourceName
		updated.Operations = facts.Operations
		updated.Lifecycle = facts.Lifecycle
		updated.Mutation = facts.Mutation
		updated.Hooks = facts.Hooks
		updated.DeleteConfirmation = facts.DeleteConfirmation
		updated.ListLookup = facts.ListLookup
		updated.Boundary.ProviderFactsOnly = true
		updated.Boundary.RepoAuthoredSpecPath = row.SpecPath
		updated.Boundary.RepoAuthoredLogicGapsPath = row.LogicPath

		if err := writeJSONFile(importPath, updated); err != nil {
			return report, err
		}
		report.ImportsRefreshed++
	}

	if _, err := RenderDiagrams(RenderOptions{
		Root:    formalRoot,
		Service: serviceFilter,
	}); err != nil {
		return report, err
	}

	if _, err := Verify(formalRoot); err != nil {
		return report, err
	}

	return report, nil
}

func analyzeProviderSource(providerRoot, revisionOverride string) (providerImportAnalysis, error) {
	modulePath, err := readModulePath(filepath.Join(providerRoot, "go.mod"))
	if err != nil {
		return providerImportAnalysis{}, err
	}

	revision := strings.TrimSpace(revisionOverride)
	if revision == "" {
		revision, err = gitRevision(providerRoot)
		if err != nil {
			return providerImportAnalysis{}, err
		}
	}

	env := append(os.Environ(), "GOWORK=off")
	cfg := &packages.Config{
		Dir:  providerRoot,
		Env:  env,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax,
	}
	loaded, err := packages.Load(cfg, "./internal/service/...")
	if err != nil {
		return providerImportAnalysis{}, err
	}
	if packages.PrintErrors(loaded) > 0 {
		return providerImportAnalysis{}, fmt.Errorf("failed to load provider packages from %s", providerRoot)
	}

	index := buildProviderIndex(loaded)
	resourceRegs := collectRegistrations(index, "RegisterResource")
	dataSourceRegs := collectRegistrations(index, "RegisterDatasource")
	listLookups, err := collectListLookups(dataSourceRegs)
	if err != nil {
		return providerImportAnalysis{}, err
	}

	resources := make(map[string]providerResourceFacts, len(resourceRegs))
	skipped := map[string]string{}
	for _, reg := range resourceRegs {
		facts, err := buildProviderResourceFacts(reg, listLookups[lookupKey(reg.Package.PkgPath, reg.Constructor)])
		if err != nil {
			skipped[reg.TerraformName] = err.Error()
			continue
		}
		resources[reg.TerraformName] = facts
	}

	return providerImportAnalysis{
		ModulePath: modulePath,
		Revision:   revision,
		Resources:  resources,
		Skipped:    skipped,
	}, nil
}

func buildProviderIndex(loaded []*packages.Package) map[string]*providerPackageContext {
	index := make(map[string]*providerPackageContext, len(loaded))
	for _, pkg := range loaded {
		ctx := &providerPackageContext{
			PkgPath: pkg.PkgPath,
			Funcs:   map[string]*providerFuncContext{},
			Methods: map[string]map[string]*providerFuncContext{},
		}
		for fileIndex, fileNode := range pkg.Syntax {
			imports := fileImportAliases(fileNode)
			_ = fileIndex
			for _, decl := range fileNode.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				fnCtx := &providerFuncContext{
					Decl:    funcDecl,
					Imports: imports,
				}
				if funcDecl.Recv == nil {
					ctx.Funcs[funcDecl.Name.Name] = fnCtx
					continue
				}
				receiverName, receiverType := receiverInfo(funcDecl)
				if receiverType == "" {
					continue
				}
				fnCtx.ReceiverName = receiverName
				if ctx.Methods[receiverType] == nil {
					ctx.Methods[receiverType] = map[string]*providerFuncContext{}
				}
				ctx.Methods[receiverType][funcDecl.Name.Name] = fnCtx
			}
		}
		index[pkg.PkgPath] = ctx
	}
	return index
}

func collectRegistrations(index map[string]*providerPackageContext, registerFunc string) []providerRegistration {
	var pkgPaths []string
	for pkgPath := range index {
		pkgPaths = append(pkgPaths, pkgPath)
	}
	sort.Strings(pkgPaths)

	var registrations []providerRegistration
	for _, pkgPath := range pkgPaths {
		pkg := index[pkgPath]
		fn := pkg.Funcs[registerFunc]
		if fn == nil || fn.Decl.Body == nil {
			continue
		}
		ast.Inspect(fn.Decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok || receiver.Name != "tfresource" || selector.Sel.Name != registerFunc {
				return true
			}
			if len(call.Args) != 2 {
				return true
			}
			terraformName, ok := stringLiteral(call.Args[0])
			if !ok {
				return true
			}
			constructorCall, ok := call.Args[1].(*ast.CallExpr)
			if !ok {
				return true
			}
			constructorName := identName(constructorCall.Fun)
			if constructorName == "" {
				return true
			}
			registrations = append(registrations, providerRegistration{
				TerraformName: terraformName,
				Constructor:   constructorName,
				Package:       pkg,
			})
			return true
		})
	}

	sort.Slice(registrations, func(i, j int) bool {
		if registrations[i].TerraformName != registrations[j].TerraformName {
			return registrations[i].TerraformName < registrations[j].TerraformName
		}
		if registrations[i].Package.PkgPath != registrations[j].Package.PkgPath {
			return registrations[i].Package.PkgPath < registrations[j].Package.PkgPath
		}
		return registrations[i].Constructor < registrations[j].Constructor
	})

	return registrations
}

func collectListLookups(registrations []providerRegistration) (map[string]listLookupFacts, error) {
	lookups := map[string]listLookupFacts{}
	for _, reg := range registrations {
		constructorFn := reg.Package.Funcs[reg.Constructor]
		if constructorFn == nil {
			return nil, fmt.Errorf("%s: missing datasource constructor %s", reg.Package.PkgPath, reg.Constructor)
		}
		resourceLit, err := resourceLiteralFromFunc(constructorFn)
		if err != nil {
			continue
		}
		schemaMap := extractSchemaMap(resourceLit)
		collectionField, resourceConstructor := findListCollection(schemaMap)
		if collectionField == "" || resourceConstructor == "" {
			continue
		}

		readHandler := resourceHandlerName(resourceLit, "read")
		readFn := reg.Package.Funcs[readHandler]
		if readFn == nil {
			return nil, fmt.Errorf("%s: datasource %s is missing read handler %s", reg.Package.PkgPath, reg.Constructor, readHandler)
		}
		crudType, _ := crudTypeFromHandler(readFn)
		methods := reg.Package.Methods[crudType]
		if methods == nil {
			return nil, fmt.Errorf("%s: datasource %s is missing CRUD methods for %s", reg.Package.PkgPath, reg.Constructor, crudType)
		}

		getClosure := methodClosure(methods, "Get")
		setDataClosure := methodClosure(methods, "SetData")

		bindings := collectOperationBindings(getClosure)
		if len(bindings) == 0 {
			return nil, fmt.Errorf("%s: datasource %s did not expose a list operation binding", reg.Package.PkgPath, reg.Constructor)
		}

		responseItemsField := findResponseItemsField(setDataClosure)
		if responseItemsField == "" {
			responseItemsField = findResponseItemsField(getClosure)
		}
		responseItemsField = normalizeResponseItemsField(responseItemsField)

		filterFields := collectTopLevelInputFields(schemaMap, collectionField)
		sort.Strings(filterFields)

		key := lookupKey(reg.Package.PkgPath, resourceConstructor)
		if _, exists := lookups[key]; exists {
			return nil, fmt.Errorf("%s: duplicate list datasource binding for %s", reg.Package.PkgPath, resourceConstructor)
		}
		lookups[key] = listLookupFacts{
			Bindings: bindings,
			Lookup: listLookup{
				Datasource:         reg.TerraformName,
				CollectionField:    collectionField,
				ResponseItemsField: responseItemsField,
				FilterFields:       filterFields,
			},
		}
	}
	return lookups, nil
}

func buildProviderResourceFacts(reg providerRegistration, listFacts listLookupFacts) (providerResourceFacts, error) {
	constructorFn := reg.Package.Funcs[reg.Constructor]
	if constructorFn == nil {
		return providerResourceFacts{}, fmt.Errorf("%s: missing resource constructor %s", reg.Package.PkgPath, reg.Constructor)
	}

	resourceLit, err := resourceLiteralFromFunc(constructorFn)
	if err != nil {
		return providerResourceFacts{}, err
	}

	createHandler := resourceHandlerName(resourceLit, "create")
	readHandler := resourceHandlerName(resourceLit, "read")
	updateHandler := resourceHandlerName(resourceLit, "update")
	deleteHandler := resourceHandlerName(resourceLit, "delete")

	crudType, createHelper := crudTypeFromHandler(reg.Package.Funcs[createHandler])
	if crudType == "" {
		crudType, _ = crudTypeFromHandler(reg.Package.Funcs[readHandler])
	}
	updateCrudType, updateHelper := crudTypeFromHandler(reg.Package.Funcs[updateHandler])
	deleteCrudType, deleteHelper := crudTypeFromHandler(reg.Package.Funcs[deleteHandler])
	if crudType == "" {
		crudType = updateCrudType
	}
	if crudType == "" {
		crudType = deleteCrudType
	}
	if crudType == "" {
		return providerResourceFacts{}, fmt.Errorf("%s: resource %s did not resolve a CRUD type", reg.Package.PkgPath, reg.Constructor)
	}

	methods := reg.Package.Methods[crudType]
	if methods == nil {
		return providerResourceFacts{}, fmt.Errorf("%s: missing methods for CRUD type %s", reg.Package.PkgPath, crudType)
	}

	createClosure := methodClosure(methods, "Create")
	getClosure := methodClosure(methods, "Get")
	updateClosure := methodClosure(methods, "Update")
	deleteClosure := methodClosure(methods, "Delete")

	schemaMap := extractSchemaMap(resourceLit)
	schemaFields := collectSchemaFields(schemaMap, "")
	forceNew, mutable, conflicts := mutationFromSchema(schemaFields)

	facts := providerResourceFacts{
		Operations: operations{
			Create: collectOperationBindings(createClosure),
			Get:    collectOperationBindings(getClosure),
			Update: collectOperationBindings(updateClosure),
			Delete: collectOperationBindings(deleteClosure),
			List:   listFacts.Bindings,
		},
		Lifecycle: lifecycle{
			Create: lifecyclePhase{
				Pending: extractLifecyclePhase(methods["CreatedPending"]),
				Target:  extractLifecyclePhase(methods["CreatedTarget"]),
			},
			Update: lifecyclePhase{
				Pending: extractLifecyclePhase(methods["UpdatedPending"]),
				Target:  extractLifecyclePhase(methods["UpdatedTarget"]),
			},
		},
		Mutation: mutation{
			Mutable:       mutable,
			ForceNew:      forceNew,
			ConflictsWith: conflicts,
		},
		Hooks: hooks{
			Create: appendGenericHook(collectHooks(createClosure), createHelper),
			Update: appendGenericHook(collectHooks(updateClosure), updateHelper),
			Delete: appendGenericHook(collectHooks(deleteClosure), deleteHelper),
		},
		DeleteConfirmation: lifecyclePhase{
			Pending: extractLifecyclePhase(methods["DeletedPending"]),
			Target:  extractLifecyclePhase(methods["DeletedTarget"]),
		},
	}
	if len(listFacts.Bindings) > 0 {
		lookupCopy := listFacts.Lookup
		facts.ListLookup = &lookupCopy
	}
	return facts, nil
}

func resourceLiteralFromFunc(fn *providerFuncContext) (*ast.CompositeLit, error) {
	if fn == nil || fn.Decl.Body == nil {
		return nil, fmt.Errorf("expected resource constructor body")
	}

	locals := map[string]ast.Expr{}
	for _, stmt := range fn.Decl.Body.List {
		switch typed := stmt.(type) {
		case *ast.AssignStmt:
			for i := range typed.Lhs {
				ident, ok := typed.Lhs[i].(*ast.Ident)
				if !ok || i >= len(typed.Rhs) {
					continue
				}
				locals[ident.Name] = typed.Rhs[i]
			}
		case *ast.DeclStmt:
			gen, ok := typed.Decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i := range valueSpec.Names {
					if i >= len(valueSpec.Values) {
						continue
					}
					locals[valueSpec.Names[i].Name] = valueSpec.Values[i]
				}
			}
		case *ast.ReturnStmt:
			if len(typed.Results) != 1 {
				continue
			}
			lit := resolveCompositeLiteral(typed.Results[0], locals)
			if lit != nil && typeName(lit.Type) == "Resource" {
				return lit, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to resolve schema.Resource literal from %s", fn.Decl.Name.Name)
}

func resolveCompositeLiteral(expr ast.Expr, locals map[string]ast.Expr) *ast.CompositeLit {
	switch typed := unparen(expr).(type) {
	case *ast.CompositeLit:
		return typed
	case *ast.UnaryExpr:
		if typed.Op == token.AND {
			if lit, ok := unparen(typed.X).(*ast.CompositeLit); ok {
				return lit
			}
		}
	case *ast.Ident:
		if localExpr, ok := locals[typed.Name]; ok {
			return resolveCompositeLiteral(localExpr, locals)
		}
	}
	return nil
}

func extractSchemaMap(resourceLit *ast.CompositeLit) map[string]ast.Expr {
	result := map[string]ast.Expr{}
	for _, elt := range resourceLit.Elts {
		keyValue, ok := elt.(*ast.KeyValueExpr)
		if !ok || identName(keyValue.Key) != "Schema" {
			continue
		}
		mapLit := resolveCompositeLiteral(keyValue.Value, nil)
		if mapLit == nil {
			if direct, ok := unparen(keyValue.Value).(*ast.CompositeLit); ok {
				mapLit = direct
			}
		}
		if mapLit == nil {
			return result
		}
		for _, schemaElt := range mapLit.Elts {
			schemaKV, ok := schemaElt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			fieldName, ok := stringLiteral(schemaKV.Key)
			if !ok {
				continue
			}
			result[fieldName] = schemaKV.Value
		}
	}
	return result
}

func findListCollection(schemaMap map[string]ast.Expr) (string, string) {
	var collectionField string
	var resourceConstructor string
	for fieldName, fieldExpr := range schemaMap {
		schemaLit := schemaLiteral(fieldExpr)
		if schemaLit == nil {
			continue
		}
		for _, elt := range schemaLit.Elts {
			keyValue, ok := elt.(*ast.KeyValueExpr)
			if !ok || identName(keyValue.Key) != "Elem" {
				continue
			}
			call, ok := unparen(keyValue.Value).(*ast.CallExpr)
			if !ok {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			pkgIdent, ok := selector.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "tfresource" || selector.Sel.Name != "GetDataSourceItemSchema" || len(call.Args) != 1 {
				continue
			}
			resourceCall, ok := call.Args[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			constructor := identName(resourceCall.Fun)
			if constructor == "" {
				continue
			}
			collectionField = fieldName
			resourceConstructor = constructor
		}
	}
	return collectionField, resourceConstructor
}

func resourceHandlerName(resourceLit *ast.CompositeLit, operation string) string {
	keys := map[string][]string{
		"create": {"Create", "CreateContext"},
		"read":   {"Read", "ReadContext"},
		"update": {"Update", "UpdateContext"},
		"delete": {"Delete", "DeleteContext"},
	}
	for _, wantKey := range keys[operation] {
		for _, elt := range resourceLit.Elts {
			keyValue, ok := elt.(*ast.KeyValueExpr)
			if !ok || identName(keyValue.Key) != wantKey {
				continue
			}
			if name := identName(keyValue.Value); name != "" {
				return name
			}
		}
	}
	return ""
}

func crudTypeFromHandler(fn *providerFuncContext) (string, string) {
	if fn == nil || fn.Decl.Body == nil {
		return "", ""
	}

	locals := map[string]string{}
	for _, stmt := range fn.Decl.Body.List {
		switch typed := stmt.(type) {
		case *ast.AssignStmt:
			for i := range typed.Lhs {
				ident, ok := typed.Lhs[i].(*ast.Ident)
				if !ok || i >= len(typed.Rhs) {
					continue
				}
				if composite := resolveCompositeLiteral(typed.Rhs[i], nil); composite != nil {
					if localType := typeName(composite.Type); localType != "" {
						locals[ident.Name] = localType
					}
				}
			}
		case *ast.ReturnStmt:
			if len(typed.Results) != 1 {
				continue
			}
			call, ok := unparen(typed.Results[0]).(*ast.CallExpr)
			if !ok {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			pkgIdent, ok := selector.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "tfresource" {
				continue
			}
			helper := pkgIdent.Name + "." + selector.Sel.Name
			for _, arg := range call.Args {
				if ident, ok := arg.(*ast.Ident); ok {
					if crudType, ok := locals[ident.Name]; ok {
						return crudType, helper
					}
				}
			}
			return "", helper
		}
	}
	return "", ""
}

func methodClosure(methods map[string]*providerFuncContext, root string) []*providerFuncContext {
	if methods == nil {
		return nil
	}
	var out []*providerFuncContext
	seen := map[string]struct{}{}
	var visit func(string)
	visit = func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		ctx := methods[name]
		if ctx == nil || ctx.Decl.Body == nil {
			return
		}
		out = append(out, ctx)
		receiverName := ctx.ReceiverName
		ast.Inspect(ctx.Decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok || receiver.Name != receiverName {
				return true
			}
			if selector.Sel.Name != root {
				if _, ok := nonRecursiveReceiverMethods[selector.Sel.Name]; ok {
					return true
				}
			}
			if _, ok := methods[selector.Sel.Name]; ok {
				visit(selector.Sel.Name)
			}
			return true
		})
	}
	visit(root)
	return out
}

func collectOperationBindings(closures []*providerFuncContext) []operationBinding {
	var bindings []operationBinding
	for _, closure := range closures {
		requestTypes := requestVars(closure)
		if len(requestTypes) == 0 || closure.Decl.Body == nil {
			continue
		}
		ast.Inspect(closure.Decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			clientSelector, ok := selector.X.(*ast.SelectorExpr)
			if !ok || clientSelector.Sel.Name != "Client" {
				return true
			}

			requestVarName := requestArgName(call.Args, requestTypes)
			if requestVarName == "" {
				return true
			}
			requestType := requestTypes[requestVarName]
			if requestType == "" {
				return true
			}
			bindings = append(bindings, operationBinding{
				Operation:    selector.Sel.Name,
				RequestType:  requestType,
				ResponseType: responseTypeFor(requestType, selector.Sel.Name),
			})
			return true
		})
	}
	return uniqueOperationBindings(bindings)
}

func requestVars(fn *providerFuncContext) map[string]string {
	requests := map[string]string{}
	if fn == nil || fn.Decl.Body == nil {
		return requests
	}
	ast.Inspect(fn.Decl.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			for i := range typed.Lhs {
				if i >= len(typed.Rhs) {
					continue
				}
				requestType := requestTypeName(typed.Rhs[i], fn.Imports)
				if requestType == "" {
					continue
				}
				ident, ok := typed.Lhs[i].(*ast.Ident)
				if !ok {
					continue
				}
				requests[ident.Name] = requestType
			}
		case *ast.ValueSpec:
			for i := range typed.Names {
				if i >= len(typed.Values) {
					continue
				}
				requestType := requestTypeName(typed.Values[i], fn.Imports)
				if requestType == "" {
					continue
				}
				requests[typed.Names[i].Name] = requestType
			}
		}
		return true
	})
	return requests
}

func requestTypeName(expr ast.Expr, imports map[string]string) string {
	composite := resolveCompositeLiteral(expr, nil)
	if composite == nil {
		return ""
	}
	typeNameValue := canonicalTypeName(composite.Type, imports)
	if !strings.HasSuffix(typeNameValue, "Request") {
		return ""
	}
	return typeNameValue
}

func collectHooks(closures []*providerFuncContext) []hook {
	var collected []hook
	for _, closure := range closures {
		if closure == nil || closure.Decl.Body == nil {
			continue
		}
		ast.Inspect(closure.Decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkgIdent, ok := selector.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "tfresource" {
				return true
			}
			helper := pkgIdent.Name + "." + selector.Sel.Name
			if !strings.HasPrefix(selector.Sel.Name, "WaitFor") && !strings.Contains(selector.Sel.Name, "WorkRequest") {
				return true
			}
			nextHook := hook{Helper: helper}
			if selector.Sel.Name == "WaitForWorkRequestWithErrorHandling" || selector.Sel.Name == "WaitForWorkRequest" {
				if len(call.Args) > 2 {
					nextHook.EntityType = enumValue(call.Args[2], closure.Imports)
				}
				if len(call.Args) > 3 {
					nextHook.Action = enumValue(call.Args[3], closure.Imports)
				}
			}
			collected = append(collected, nextHook)
			return true
		})
	}
	return uniqueHooks(collected)
}

func appendGenericHook(existing []hook, helper string) []hook {
	switch helper {
	case "tfresource.CreateResource", "tfresource.CreateResourceWithContext", "tfresource.UpdateResource", "tfresource.UpdateResourceWithContext", "tfresource.DeleteResource", "tfresource.DeleteResourceWithContext":
		existing = append(existing, hook{Helper: helper})
	}
	return uniqueHooks(existing)
}

func collectSchemaFields(schemaMap map[string]ast.Expr, prefix string) []schemaField {
	var out []schemaField
	var fieldNames []string
	for fieldName := range schemaMap {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		fieldExpr := schemaMap[fieldName]
		fieldLit := schemaLiteral(fieldExpr)
		if fieldLit == nil {
			continue
		}
		fieldPath := joinPath(prefix, fieldName)
		definition := schemaField{Path: fieldPath}
		nestedSchema := map[string]ast.Expr{}
		for _, elt := range fieldLit.Elts {
			keyValue, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			switch identName(keyValue.Key) {
			case "Optional":
				definition.Optional = boolValue(keyValue.Value)
			case "Required":
				definition.Required = boolValue(keyValue.Value)
			case "Computed":
				definition.Computed = boolValue(keyValue.Value)
			case "ForceNew":
				definition.ForceNew = boolValue(keyValue.Value)
			case "ConflictsWith":
				definition.ConflictsWith = stringSlice(keyValue.Value)
			case "Elem":
				nestedSchema = nestedSchemaMap(keyValue.Value)
			}
		}

		recordSelf := definition.Path != "" && (len(nestedSchema) == 0 || definition.ForceNew || len(definition.ConflictsWith) > 0)
		if recordSelf {
			sort.Strings(definition.ConflictsWith)
			out = append(out, definition)
		}
		if len(nestedSchema) > 0 {
			out = append(out, collectSchemaFields(nestedSchema, fieldPath)...)
		}
	}
	return out
}

func nestedSchemaMap(expr ast.Expr) map[string]ast.Expr {
	nestedLit := resolveCompositeLiteral(expr, nil)
	if nestedLit == nil || typeName(nestedLit.Type) != "Resource" {
		return nil
	}
	return extractSchemaMap(nestedLit)
}

func mutationFromSchema(fields []schemaField) ([]string, []string, map[string][]string) {
	forceNew := []string{}
	mutable := []string{}
	conflicts := map[string][]string{}
	for _, field := range fields {
		if field.ForceNew {
			forceNew = append(forceNew, field.Path)
			continue
		}
		if field.Required || field.Optional {
			mutable = append(mutable, field.Path)
		}
		if len(field.ConflictsWith) > 0 {
			conflicts[field.Path] = uniqueSortedStrings(field.ConflictsWith)
		}
	}
	sort.Strings(forceNew)
	sort.Strings(mutable)
	return uniqueSortedStrings(forceNew), uniqueSortedStrings(mutable), conflicts
}

func collectTopLevelInputFields(schemaMap map[string]ast.Expr, collectionField string) []string {
	var inputs []string
	for fieldName, fieldExpr := range schemaMap {
		if fieldName == collectionField || fieldName == "filter" {
			continue
		}
		fieldLit := schemaLiteral(fieldExpr)
		if fieldLit == nil {
			continue
		}
		var optional bool
		var required bool
		for _, elt := range fieldLit.Elts {
			keyValue, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			switch identName(keyValue.Key) {
			case "Optional":
				optional = boolValue(keyValue.Value)
			case "Required":
				required = boolValue(keyValue.Value)
			}
		}
		if optional || required {
			inputs = append(inputs, fieldName)
		}
	}
	return uniqueSortedStrings(inputs)
}

func extractLifecyclePhase(fn *providerFuncContext) []string {
	if fn == nil || fn.Decl.Body == nil {
		return nil
	}
	for _, stmt := range fn.Decl.Body.List {
		returnStmt, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(returnStmt.Results) != 1 {
			continue
		}
		values := stringSliceWithEnums(returnStmt.Results[0], fn.Imports)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func findResponseItemsField(closures []*providerFuncContext) string {
	for _, closure := range closures {
		if closure == nil || closure.Decl.Body == nil {
			continue
		}
		var field string
		ast.Inspect(closure.Decl.Body, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.RangeStmt:
				if field = responseFieldName(typed.X); field != "" {
					return false
				}
			case *ast.AssignStmt:
				for _, lhs := range typed.Lhs {
					if field = responseFieldName(lhs); field != "" {
						return false
					}
				}
			}
			return true
		})
		if field != "" {
			return field
		}
	}
	return ""
}

func normalizeResponseItemsField(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return defaultResponseItemsField
	}
	return field
}

func lookupKey(pkgPath, constructor string) string {
	return pkgPath + "::" + constructor
}

func resolveProviderRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			if err := requireDirectory(filepath.Join(current, "internal", "service")); err != nil {
				return "", err
			}
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("unable to locate provider module root from %s", start)
		}
		current = parent
	}
}

func readModulePath(goModPath string) (string, error) {
	contents, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("%s: missing module directive", filepath.ToSlash(goModPath))
}

func gitRevision(root string) (string, error) {
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve provider revision: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func upsertSource(existing []sourceLockEntry, replacement sourceLockEntry) []sourceLockEntry {
	out := make([]sourceLockEntry, 0, len(existing)+1)
	replaced := false
	for _, source := range existing {
		if source.Name == replacement.Name {
			out = append(out, replacement)
			replaced = true
			continue
		}
		out = append(out, source)
	}
	if !replaced {
		out = append(out, replacement)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func writeJSONFile(path string, value any) error {
	contents, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')

	current, err := os.ReadFile(path)
	if err == nil && bytes.Equal(current, contents) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, contents, 0o644)
}

func fileImportAliases(file *ast.File) map[string]string {
	imports := map[string]string{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		alias := path.Base(importPath)
		if spec.Name != nil && spec.Name.Name != "" {
			alias = spec.Name.Name
		}
		imports[alias] = importPath
	}
	return imports
}

func receiverInfo(fn *ast.FuncDecl) (string, string) {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return "", ""
	}
	receiverName := ""
	if len(fn.Recv.List[0].Names) > 0 {
		receiverName = fn.Recv.List[0].Names[0].Name
	}
	return receiverName, typeName(fn.Recv.List[0].Type)
}

func typeName(expr ast.Expr) string {
	switch typed := unparen(expr).(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	case *ast.StarExpr:
		return typeName(typed.X)
	}
	return ""
}

func canonicalTypeName(expr ast.Expr, imports map[string]string) string {
	switch typed := unparen(expr).(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		if ident, ok := typed.X.(*ast.Ident); ok {
			if importPath, ok := imports[ident.Name]; ok {
				return path.Base(importPath) + "." + typed.Sel.Name
			}
			return ident.Name + "." + typed.Sel.Name
		}
	}
	return ""
}

func schemaLiteral(expr ast.Expr) *ast.CompositeLit {
	lit := resolveCompositeLiteral(expr, nil)
	if lit == nil {
		return nil
	}
	if lit.Type == nil || typeName(lit.Type) == "Schema" {
		return lit
	}
	return nil
}

func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}

func unparen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

func identName(expr ast.Expr) string {
	if ident, ok := unparen(expr).(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := unparen(expr).(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func boolValue(expr ast.Expr) bool {
	ident, ok := unparen(expr).(*ast.Ident)
	return ok && ident.Name == "true"
}

func stringSlice(expr ast.Expr) []string {
	composite := resolveCompositeLiteral(expr, nil)
	if composite == nil {
		return nil
	}
	var values []string
	for _, elt := range composite.Elts {
		if value, ok := stringLiteral(elt); ok {
			values = append(values, value)
		}
	}
	return uniqueSortedStrings(values)
}

func stringSliceWithEnums(expr ast.Expr, imports map[string]string) []string {
	composite := resolveCompositeLiteral(expr, nil)
	if composite == nil {
		return nil
	}
	var values []string
	for _, elt := range composite.Elts {
		if value := enumValue(elt, imports); value != "" {
			values = append(values, value)
		}
	}
	return uniqueSortedStrings(values)
}

func enumValue(expr ast.Expr, imports map[string]string) string {
	switch typed := unparen(expr).(type) {
	case *ast.BasicLit:
		if typed.Kind == token.STRING {
			value, ok := stringLiteral(typed)
			if ok {
				return value
			}
		}
	case *ast.CallExpr:
		if ident, ok := typed.Fun.(*ast.Ident); ok && ident.Name == "string" && len(typed.Args) == 1 {
			return enumValue(typed.Args[0], imports)
		}
	case *ast.SelectorExpr:
		return selectorEnumValue(typed, imports)
	case *ast.Ident:
		return camelToEnum(typed.Name)
	}
	return ""
}

func selectorEnumValue(expr *ast.SelectorExpr, imports map[string]string) string {
	name := expr.Sel.Name
	for _, marker := range []string{
		"WorkRequestResourceActionType",
		"LifecycleState",
		"WorkRequestStatus",
		"ActionType",
		"Status",
		"State",
	} {
		if idx := strings.LastIndex(name, marker); idx >= 0 && idx+len(marker) < len(name) {
			return camelToEnum(name[idx+len(marker):])
		}
	}
	canonical := canonicalTypeName(expr, imports)
	if strings.Contains(canonical, ".") {
		suffix := canonical[strings.LastIndex(canonical, ".")+1:]
		return camelToEnum(suffix)
	}
	return camelToEnum(name)
}

func camelToEnum(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsUpper(r) && (unicode.IsLower(prev) || (next != 0 && unicode.IsLower(next))) {
				b.WriteByte('_')
			} else if unicode.IsDigit(r) && !unicode.IsDigit(prev) {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

func requestArgName(args []ast.Expr, requests map[string]string) string {
	for _, arg := range args {
		switch typed := unparen(arg).(type) {
		case *ast.Ident:
			if _, ok := requests[typed.Name]; ok {
				return typed.Name
			}
		case *ast.UnaryExpr:
			if typed.Op != token.AND {
				continue
			}
			if ident, ok := unparen(typed.X).(*ast.Ident); ok {
				if _, ok := requests[ident.Name]; ok {
					return ident.Name
				}
			}
		}
	}
	return ""
}

func responseTypeFor(requestType, operation string) string {
	prefix := ""
	if idx := strings.LastIndex(requestType, "."); idx >= 0 {
		prefix = requestType[:idx+1]
	}
	if operation == "" {
		return strings.TrimSuffix(requestType, "Request") + "Response"
	}
	return prefix + operation + "Response"
}

func responseFieldName(expr ast.Expr) string {
	parts := selectorParts(expr)
	if len(parts) < 2 {
		return ""
	}
	foundRes := false
	for _, part := range parts {
		if part == "Res" {
			foundRes = true
			break
		}
	}
	if !foundRes {
		return ""
	}
	leaf := parts[len(parts)-1]
	if leaf == "OpcNextPage" {
		return ""
	}
	return leaf
}

func selectorParts(expr ast.Expr) []string {
	switch typed := unparen(expr).(type) {
	case *ast.Ident:
		return []string{typed.Name}
	case *ast.SelectorExpr:
		return append(selectorParts(typed.X), typed.Sel.Name)
	}
	return nil
}

func uniqueOperationBindings(bindings []operationBinding) []operationBinding {
	if len(bindings) == 0 {
		return nil
	}
	seen := map[string]operationBinding{}
	for _, binding := range bindings {
		key := binding.Operation + "|" + binding.RequestType + "|" + binding.ResponseType
		seen[key] = binding
	}
	out := make([]operationBinding, 0, len(seen))
	for _, binding := range seen {
		out = append(out, binding)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Operation != out[j].Operation {
			return out[i].Operation < out[j].Operation
		}
		if out[i].RequestType != out[j].RequestType {
			return out[i].RequestType < out[j].RequestType
		}
		return out[i].ResponseType < out[j].ResponseType
	})
	return out
}

func uniqueHooks(hooksIn []hook) []hook {
	if len(hooksIn) == 0 {
		return nil
	}
	seen := map[string]hook{}
	for _, nextHook := range hooksIn {
		key := nextHook.Helper + "|" + nextHook.EntityType + "|" + nextHook.Action
		seen[key] = nextHook
	}
	out := make([]hook, 0, len(seen))
	for _, nextHook := range seen {
		out = append(out, nextHook)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Helper != out[j].Helper {
			return out[i].Helper < out[j].Helper
		}
		if out[i].EntityType != out[j].EntityType {
			return out[i].EntityType < out[j].EntityType
		}
		return out[i].Action < out[j].Action
	})
	return out
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
