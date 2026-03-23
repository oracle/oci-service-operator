package ocisdk

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type ResolveDirFunc func(context.Context, string) (string, error)

type FieldKind string

const (
	FieldKindScalar    FieldKind = "scalar"
	FieldKindStruct    FieldKind = "struct"
	FieldKindInterface FieldKind = "interface"
)

type Field struct {
	Name           string
	Type           string
	JSONName       string
	RenderableType string
	Mandatory      bool
	Deprecated     bool
	ReadOnly       bool
	Documentation  string
	Kind           FieldKind
	NestedFields   []Field
}

type Struct struct {
	Name   string
	Fields []Field
}

type InterfaceFamily struct {
	Base            Struct
	Implementations []Struct
}

type OperationMethod struct {
	Verb         string
	MethodName   string
	ClientType   string
	RequestType  string
	ResponseType string
	UsesRequest  bool
}

type ConstructorKind string

const (
	ConstructorKindProvider         ConstructorKind = "provider"
	ConstructorKindProviderEndpoint ConstructorKind = "provider_endpoint"
	ConstructorKindUnknown          ConstructorKind = "unknown"
)

type ClientConstructor struct {
	Name       string
	ClientType string
	Kind       ConstructorKind
}

type Index struct {
	resolveDir ResolveDirFunc

	mu       sync.Mutex
	packages map[string]*Package
}

type Package struct {
	typeNames           []string
	structs             map[string]structDefinition
	aliases             map[string]ast.Expr
	interfaces          map[string]struct{}
	polymorphic         map[string][]string
	requestBodyPayloads map[string][]string
	requestMethods      map[string]OperationMethod
	clientConstructors  map[string]ClientConstructor

	mu       sync.Mutex
	resolved map[string]Struct
}

type structDefinition struct {
	Fields []fieldDefinition
}

type fieldDefinition struct {
	Name          string
	TypeExpr      ast.Expr
	JSONName      string
	Mandatory     bool
	Documentation string
	Deprecated    bool
	ReadOnly      bool
}

var crudMethodPattern = regexp.MustCompile(`^(Create|Get|List|Update|Delete)(.+)$`)
var crudRequestPattern = regexp.MustCompile(`^(Create|Get|List|Update|Delete)(.+)Request$`)
var constructorPattern = regexp.MustCompile(`^New(.+)WithConfigurationProvider$`)

func NewIndex(resolveDir ResolveDirFunc) *Index {
	return &Index{
		resolveDir: resolveDir,
		packages:   make(map[string]*Package),
	}
}

func (index *Index) Package(ctx context.Context, importPath string) (*Package, error) {
	index.mu.Lock()
	pkg, ok := index.packages[importPath]
	index.mu.Unlock()
	if ok {
		return pkg, nil
	}
	if index.resolveDir == nil {
		return nil, fmt.Errorf("no SDK package resolver configured")
	}

	dir, err := index.resolveDir(ctx, importPath)
	if err != nil {
		return nil, err
	}

	pkg, err = parsePackage(dir)
	if err != nil {
		return nil, err
	}

	index.mu.Lock()
	defer index.mu.Unlock()
	if existing, ok := index.packages[importPath]; ok {
		return existing, nil
	}
	index.packages[importPath] = pkg
	return pkg, nil
}

func (index *Index) Struct(ctx context.Context, importPath string, typeName string) (Struct, bool, error) {
	pkg, err := index.Package(ctx, importPath)
	if err != nil {
		return Struct{}, false, err
	}
	structDef, ok := pkg.Struct(typeName)
	return structDef, ok, nil
}

func (pkg *Package) TypeNames() []string {
	return append([]string(nil), pkg.typeNames...)
}

func (pkg *Package) RequestBodyPayloads(typeName string) []string {
	return append([]string(nil), pkg.requestBodyPayloads[typeName]...)
}

func (pkg *Package) OperationForRequest(typeName string) (OperationMethod, bool) {
	method, ok := pkg.requestMethods[typeName]
	return method, ok
}

func (pkg *Package) ClientConstructor(clientType string) (ClientConstructor, bool) {
	constructor, ok := pkg.clientConstructors[clientType]
	return constructor, ok
}

func (pkg *Package) ResourceOperations(rawName string) map[string]OperationMethod {
	operations := make(map[string]OperationMethod)
	for _, method := range pkg.requestMethods {
		matches := crudRequestPattern.FindStringSubmatch(method.RequestType)
		if len(matches) == 0 {
			continue
		}
		if sdkSingularize(matches[2]) != rawName {
			continue
		}
		operations[matches[1]] = method
	}
	return operations
}

func (pkg *Package) Struct(typeName string) (Struct, bool) {
	if resolved, ok := pkg.cachedStruct(typeName); ok {
		return cloneStruct(resolved), true
	}
	if _, ok := pkg.structs[typeName]; !ok {
		return Struct{}, false
	}

	resolved := pkg.resolveStruct(typeName, map[string]struct{}{})
	pkg.mu.Lock()
	if existing, ok := pkg.resolved[typeName]; ok {
		resolved = existing
	} else {
		pkg.resolved[typeName] = resolved
	}
	pkg.mu.Unlock()

	return cloneStruct(resolved), true
}

func (pkg *Package) InterfaceFamily(typeName string) (InterfaceFamily, bool) {
	baseName := strings.ToLower(strings.TrimSpace(typeName))
	implementationNames, ok := pkg.polymorphic[baseName]
	if !ok {
		return InterfaceFamily{}, false
	}

	family := InterfaceFamily{}
	if base, ok := pkg.Struct(baseName); ok {
		family.Base = base
	}
	for _, implementationName := range implementationNames {
		implementation, ok := pkg.Struct(implementationName)
		if !ok {
			continue
		}
		family.Implementations = append(family.Implementations, implementation)
	}
	if len(family.Base.Fields) == 0 && len(family.Implementations) == 0 {
		return InterfaceFamily{}, false
	}
	return family, true
}

func (pkg *Package) cachedStruct(typeName string) (Struct, bool) {
	pkg.mu.Lock()
	defer pkg.mu.Unlock()
	resolved, ok := pkg.resolved[typeName]
	return resolved, ok
}

func (pkg *Package) resolveStruct(typeName string, visiting map[string]struct{}) Struct {
	if resolved, ok := pkg.cachedStruct(typeName); ok {
		return resolved
	}

	definition, ok := pkg.structs[typeName]
	if !ok {
		return Struct{}
	}

	visiting[typeName] = struct{}{}
	defer delete(visiting, typeName)

	resolved := Struct{
		Name:   typeName,
		Fields: make([]Field, 0, len(definition.Fields)),
	}
	for _, fieldDef := range definition.Fields {
		kind, nestedType := pkg.fieldKind(fieldDef.TypeExpr)
		renderableType, _ := pkg.renderableType(fieldDef.TypeExpr)
		field := Field{
			Name:           fieldDef.Name,
			Type:           exprString(fieldDef.TypeExpr),
			JSONName:       fieldDef.JSONName,
			RenderableType: renderableType,
			Mandatory:      fieldDef.Mandatory,
			Deprecated:     fieldDef.Deprecated,
			ReadOnly:       fieldDef.ReadOnly,
			Documentation:  fieldDef.Documentation,
			Kind:           kind,
		}
		if kind == FieldKindStruct {
			if _, seen := visiting[nestedType]; !seen {
				if nested, ok := pkg.resolveKnownStruct(nestedType, visiting); ok {
					field.NestedFields = cloneFields(nested.Fields)
				}
			}
		}
		resolved.Fields = append(resolved.Fields, field)
	}

	return resolved
}

func (pkg *Package) resolveKnownStruct(typeName string, visiting map[string]struct{}) (Struct, bool) {
	if resolved, ok := pkg.cachedStruct(typeName); ok {
		return resolved, true
	}
	if _, ok := pkg.structs[typeName]; !ok {
		return Struct{}, false
	}

	resolved := pkg.resolveStruct(typeName, visiting)
	pkg.mu.Lock()
	if existing, ok := pkg.resolved[typeName]; ok {
		resolved = existing
	} else {
		pkg.resolved[typeName] = resolved
	}
	pkg.mu.Unlock()

	return resolved, true
}

func (pkg *Package) fieldKind(expr ast.Expr) (FieldKind, string) {
	if pkg.isInterfaceExpr(expr, map[string]struct{}{}) {
		return FieldKindInterface, ""
	}
	if nestedType, ok := pkg.nestedStructTypeName(expr, map[string]struct{}{}); ok {
		return FieldKindStruct, nestedType
	}
	return FieldKindScalar, ""
}

func (pkg *Package) isInterfaceExpr(expr ast.Expr, visiting map[string]struct{}) bool {
	switch concrete := expr.(type) {
	case *ast.StarExpr:
		return pkg.isInterfaceExpr(concrete.X, visiting)
	case *ast.ArrayType:
		return pkg.isInterfaceExpr(concrete.Elt, visiting)
	case *ast.MapType:
		return pkg.isInterfaceExpr(concrete.Value, visiting)
	case *ast.Ident:
		if _, ok := pkg.interfaces[concrete.Name]; ok {
			return true
		}
		if _, seen := visiting[concrete.Name]; seen {
			return false
		}
		aliasExpr, ok := pkg.aliases[concrete.Name]
		if !ok {
			return false
		}
		visiting[concrete.Name] = struct{}{}
		defer delete(visiting, concrete.Name)
		return pkg.isInterfaceExpr(aliasExpr, visiting)
	default:
		return false
	}
}

func (pkg *Package) referencedTypeName(expr ast.Expr, visiting map[string]struct{}) (string, bool) {
	switch concrete := expr.(type) {
	case *ast.StarExpr:
		return pkg.referencedTypeName(concrete.X, visiting)
	case *ast.ArrayType:
		return pkg.referencedTypeName(concrete.Elt, visiting)
	case *ast.MapType:
		return pkg.referencedTypeName(concrete.Value, visiting)
	case *ast.Ident:
		if _, ok := pkg.structs[concrete.Name]; ok {
			return concrete.Name, true
		}
		if _, ok := pkg.interfaces[concrete.Name]; ok {
			return concrete.Name, true
		}
		if _, seen := visiting[concrete.Name]; seen {
			return "", false
		}
		aliasExpr, ok := pkg.aliases[concrete.Name]
		if !ok {
			return "", false
		}
		visiting[concrete.Name] = struct{}{}
		defer delete(visiting, concrete.Name)
		return pkg.referencedTypeName(aliasExpr, visiting)
	default:
		return "", false
	}
}

func (pkg *Package) nestedStructTypeName(expr ast.Expr, visiting map[string]struct{}) (string, bool) {
	switch concrete := expr.(type) {
	case *ast.StarExpr:
		return pkg.nestedStructTypeName(concrete.X, visiting)
	case *ast.ArrayType:
		return pkg.nestedStructTypeName(concrete.Elt, visiting)
	case *ast.MapType:
		return pkg.nestedStructTypeName(concrete.Value, visiting)
	case *ast.Ident:
		if _, ok := pkg.structs[concrete.Name]; ok {
			return concrete.Name, true
		}
		if _, seen := visiting[concrete.Name]; seen {
			return "", false
		}
		aliasExpr, ok := pkg.aliases[concrete.Name]
		if !ok {
			return "", false
		}
		visiting[concrete.Name] = struct{}{}
		defer delete(visiting, concrete.Name)
		return pkg.nestedStructTypeName(aliasExpr, visiting)
	default:
		return "", false
	}
}

func (pkg *Package) renderableType(expr ast.Expr) (string, bool) {
	return pkg.renderableTypeExpr(expr, map[string]struct{}{})
}

func (pkg *Package) renderableTypeExpr(expr ast.Expr, visiting map[string]struct{}) (string, bool) {
	switch concrete := expr.(type) {
	case *ast.Ident:
		switch concrete.Name {
		case "string", "bool", "int", "int32", "int64", "float32", "float64":
			return concrete.Name, true
		}
		if _, seen := visiting[concrete.Name]; seen {
			return "", false
		}
		aliasExpr, ok := pkg.aliases[concrete.Name]
		if !ok {
			return "", false
		}
		visiting[concrete.Name] = struct{}{}
		defer delete(visiting, concrete.Name)
		return pkg.renderableTypeExpr(aliasExpr, visiting)
	case *ast.StarExpr:
		return pkg.renderableTypeExpr(concrete.X, visiting)
	case *ast.ArrayType:
		if ident, ok := concrete.Elt.(*ast.Ident); ok && ident.Name == "byte" {
			// OCI SDK byte slices are serialized as base64-encoded JSON strings.
			return "string", true
		}
		elementType, ok := pkg.renderableTypeExpr(concrete.Elt, visiting)
		if !ok {
			return "", false
		}
		return "[]" + elementType, true
	case *ast.MapType:
		keyType, ok := pkg.renderableTypeExpr(concrete.Key, visiting)
		if !ok || keyType != "string" {
			return "", false
		}
		valueType, ok := pkg.renderableTypeExpr(concrete.Value, visiting)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("map[%s]%s", keyType, valueType), true
	case *ast.SelectorExpr:
		switch concrete.Sel.Name {
		case "SDKDate", "SDKTime", "Time":
			return "string", true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

func parsePackage(dir string) (*Package, error) {
	pkgs, err := parser.ParseDir(token.NewFileSet(), dir, func(info os.FileInfo) bool {
		return !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse dir %q: %w", dir, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go package found in %q", dir)
	}

	pkg := &Package{
		structs:             make(map[string]structDefinition),
		aliases:             make(map[string]ast.Expr),
		interfaces:          make(map[string]struct{}),
		polymorphic:         make(map[string][]string),
		requestBodyPayloads: make(map[string][]string),
		requestMethods:      make(map[string]OperationMethod),
		clientConstructors:  make(map[string]ClientConstructor),
		resolved:            make(map[string]Struct),
	}
	exportedTypes := make(map[string]struct{})
	requestBodyPayloadExprs := make(map[string][]ast.Expr)
	for _, parsedPackage := range pkgs {
		for _, fileNode := range parsedPackage.Files {
			for _, declaration := range fileNode.Decls {
				switch typedDeclaration := declaration.(type) {
				case *ast.GenDecl:
					if typedDeclaration.Tok != token.TYPE {
						continue
					}
					for _, spec := range typedDeclaration.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						if typeSpec.Name.IsExported() {
							exportedTypes[typeSpec.Name.Name] = struct{}{}
						}

						switch concrete := typeSpec.Type.(type) {
						case *ast.StructType:
							pkg.structs[typeSpec.Name.Name] = parseStruct(concrete)
							requestBodyPayloadExprs[typeSpec.Name.Name] = append(requestBodyPayloadExprs[typeSpec.Name.Name], bodyContributorExprs(concrete)...)
						case *ast.InterfaceType:
							pkg.interfaces[typeSpec.Name.Name] = struct{}{}
						default:
							pkg.aliases[typeSpec.Name.Name] = concrete
						}
					}
				case *ast.FuncDecl:
					receiverName, implementations := polymorphicMethod(typedDeclaration)
					if receiverName != "" && len(implementations) > 0 {
						pkg.polymorphic[receiverName] = appendUniqueNames(pkg.polymorphic[receiverName], implementations...)
					}
					if method, ok := operationMethod(typedDeclaration); ok {
						pkg.requestMethods[method.RequestType] = method
					}
					if constructor, ok := clientConstructor(typedDeclaration); ok {
						pkg.clientConstructors[constructor.ClientType] = constructor
					}
				}
			}
		}
	}

	for typeName, exprs := range requestBodyPayloadExprs {
		for _, expr := range exprs {
			payloadType, ok := pkg.referencedTypeName(expr, map[string]struct{}{})
			if !ok {
				continue
			}
			pkg.requestBodyPayloads[typeName] = appendUniqueNames(pkg.requestBodyPayloads[typeName], payloadType)
		}
	}

	for typeName := range exportedTypes {
		pkg.typeNames = append(pkg.typeNames, typeName)
	}
	sort.Strings(pkg.typeNames)

	return pkg, nil
}

func parseStruct(structType *ast.StructType) structDefinition {
	definition := structDefinition{}
	if structType.Fields == nil {
		return definition
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		jsonName, ok := jsonFieldName(field.Tag)
		if !ok {
			continue
		}
		documentation := fieldDocumentation(field)
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			definition.Fields = append(definition.Fields, fieldDefinition{
				Name:          name.Name,
				TypeExpr:      field.Type,
				JSONName:      jsonName,
				Mandatory:     mandatoryField(field.Tag),
				Documentation: documentation,
				Deprecated:    strings.Contains(strings.ToLower(documentation), "deprecated"),
				ReadOnly:      isReadOnlyDocumentation(documentation),
			})
		}
	}

	return definition
}

func bodyContributorExprs(structType *ast.StructType) []ast.Expr {
	if structType.Fields == nil {
		return nil
	}

	contributors := make([]ast.Expr, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		if !contributesToBody(field.Tag) {
			continue
		}
		contributors = append(contributors, field.Type)
	}

	return contributors
}

func contributesToBody(tag *ast.BasicLit) bool {
	structTag, ok := parseStructTag(tag)
	if !ok {
		return false
	}
	return structTag.Get("contributesTo") == "body"
}

func jsonFieldName(tag *ast.BasicLit) (string, bool) {
	structTag, ok := parseStructTag(tag)
	if !ok {
		return "", false
	}
	if structTag == "" {
		return "", true
	}
	jsonTag := structTag.Get("json")
	if jsonTag == "-" {
		return "", false
	}
	if jsonTag == "" {
		return "", true
	}

	name := strings.Split(jsonTag, ",")[0]
	if name == "-" {
		return "", false
	}
	return name, true
}

func mandatoryField(tag *ast.BasicLit) bool {
	structTag, ok := parseStructTag(tag)
	if !ok {
		return false
	}
	return structTag.Get("mandatory") == "true"
}

func parseStructTag(tag *ast.BasicLit) (reflect.StructTag, bool) {
	if tag == nil {
		return "", true
	}

	unquoted, err := strconv.Unquote(tag.Value)
	if err != nil {
		return "", false
	}
	return reflect.StructTag(unquoted), true
}

func fieldDocumentation(field *ast.Field) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}
	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}
	return ""
}

func isReadOnlyDocumentation(documentation string) bool {
	lower := strings.ToLower(documentation)
	return strings.Contains(lower, "read-only") ||
		strings.Contains(lower, "read only") ||
		strings.Contains(lower, "server-side") ||
		strings.Contains(lower, "server side") ||
		strings.Contains(lower, "output only")
}

func exprString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, token.NewFileSet(), expr); err != nil {
		return ""
	}
	return buffer.String()
}

func cloneStruct(in Struct) Struct {
	return Struct{
		Name:   in.Name,
		Fields: cloneFields(in.Fields),
	}
}

func cloneFields(fields []Field) []Field {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]Field, len(fields))
	for i, field := range fields {
		cloned[i] = field
		cloned[i].NestedFields = cloneFields(field.NestedFields)
	}
	return cloned
}

func polymorphicMethod(decl *ast.FuncDecl) (string, []string) {
	if decl == nil || decl.Name == nil || decl.Name.Name != "UnmarshalPolymorphicJSON" || decl.Recv == nil || decl.Body == nil || len(decl.Recv.List) == 0 {
		return "", nil
	}

	receiverName := receiverTypeName(decl.Recv.List[0].Type)
	if receiverName == "" {
		return "", nil
	}

	var implementations []string
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		switchStmt, ok := node.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		for _, statement := range switchStmt.Body.List {
			caseClause, ok := statement.(*ast.CaseClause)
			if !ok || len(caseClause.List) == 0 {
				continue
			}
			if implementation := caseImplementation(caseClause.Body); implementation != "" {
				implementations = append(implementations, implementation)
			}
		}
		return false
	})

	return receiverName, appendUniqueNames(nil, implementations...)
}

func receiverTypeName(expr ast.Expr) string {
	switch concrete := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(concrete.X)
	case *ast.Ident:
		return concrete.Name
	default:
		return ""
	}
}

func caseImplementation(statements []ast.Stmt) string {
	for _, statement := range statements {
		implementation := ""
		ast.Inspect(statement, func(node ast.Node) bool {
			if implementation != "" {
				return false
			}
			compositeLiteral, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			ident, ok := compositeLiteral.Type.(*ast.Ident)
			if !ok {
				return true
			}
			implementation = ident.Name
			return false
		})
		if implementation != "" {
			return implementation
		}
	}
	return ""
}

func appendUniqueNames(existing []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, name := range existing {
		seen[name] = struct{}{}
	}
	for _, name := range extras {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		existing = append(existing, name)
	}
	return existing
}

func clientConstructor(decl *ast.FuncDecl) (ClientConstructor, bool) {
	if decl == nil || decl.Name == nil || decl.Recv != nil || decl.Type == nil || decl.Type.Params == nil || decl.Type.Results == nil {
		return ClientConstructor{}, false
	}

	matches := constructorPattern.FindStringSubmatch(decl.Name.Name)
	if len(matches) == 0 {
		return ClientConstructor{}, false
	}

	clientType := matches[1]
	resultType, ok := firstNamedType(decl.Type.Results.List)
	if !ok || resultType != clientType {
		return ClientConstructor{}, false
	}

	return ClientConstructor{
		Name:       decl.Name.Name,
		ClientType: clientType,
		Kind:       constructorKind(decl.Type.Params.List),
	}, true
}

func constructorKind(fields []*ast.Field) ConstructorKind {
	params := parameterExprs(fields)
	switch {
	case len(params) == 1 && isConfigurationProviderType(params[0]):
		return ConstructorKindProvider
	case len(params) == 2 && isConfigurationProviderType(params[0]) && isStringType(params[1]):
		return ConstructorKindProviderEndpoint
	default:
		return ConstructorKindUnknown
	}
}

func parameterExprs(fields []*ast.Field) []ast.Expr {
	params := make([]ast.Expr, 0, len(fields))
	for _, field := range fields {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			params = append(params, field.Type)
		}
	}
	return params
}

func isConfigurationProviderType(expr ast.Expr) bool {
	typeName, ok := namedType(expr)
	return ok && typeName == "ConfigurationProvider"
}

func isStringType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

func operationMethod(decl *ast.FuncDecl) (OperationMethod, bool) {
	if decl == nil || decl.Name == nil || decl.Recv == nil || decl.Type == nil || decl.Type.Params == nil || decl.Type.Results == nil {
		return OperationMethod{}, false
	}

	matches := crudMethodPattern.FindStringSubmatch(decl.Name.Name)
	if len(matches) == 0 {
		return OperationMethod{}, false
	}

	if len(decl.Recv.List) == 0 {
		return OperationMethod{}, false
	}
	clientType := receiverTypeName(decl.Recv.List[0].Type)
	if clientType == "" || !strings.HasSuffix(clientType, "Client") {
		return OperationMethod{}, false
	}
	if !hasContextParam(decl.Type.Params.List) {
		return OperationMethod{}, false
	}

	requestType, hasRequest := firstNamedTypeAfterContext(decl.Type.Params.List)
	if !hasRequest {
		requestType = decl.Name.Name + "Request"
	}
	responseType, ok := firstNamedType(decl.Type.Results.List)
	if !ok {
		return OperationMethod{}, false
	}
	if requestType != decl.Name.Name+"Request" || responseType != decl.Name.Name+"Response" {
		return OperationMethod{}, false
	}

	return OperationMethod{
		Verb:         matches[1],
		MethodName:   decl.Name.Name,
		ClientType:   clientType,
		RequestType:  requestType,
		ResponseType: responseType,
		UsesRequest:  hasRequest,
	}, true
}

func firstNamedTypeAfterContext(fields []*ast.Field) (string, bool) {
	contextSeen := false
	for _, field := range fields {
		typeName, ok := namedType(field.Type)
		if !ok {
			continue
		}
		if typeName == "Context" {
			contextSeen = true
			continue
		}
		if contextSeen {
			return typeName, true
		}
	}
	return "", false
}

func hasContextParam(fields []*ast.Field) bool {
	for _, field := range fields {
		typeName, ok := namedType(field.Type)
		if ok && typeName == "Context" {
			return true
		}
	}
	return false
}

func firstNamedType(fields []*ast.Field) (string, bool) {
	for _, field := range fields {
		typeName, ok := namedType(field.Type)
		if ok {
			return typeName, true
		}
	}
	return "", false
}

func namedType(expr ast.Expr) (string, bool) {
	switch concrete := expr.(type) {
	case *ast.Ident:
		return concrete.Name, true
	case *ast.SelectorExpr:
		return concrete.Sel.Name, true
	case *ast.StarExpr:
		return namedType(concrete.X)
	default:
		return "", false
	}
}

func sdkSingularize(name string) string {
	switch {
	case strings.HasSuffix(name, "Statuses") && len(name) > len("Statuses"):
		return strings.TrimSuffix(name, "Statuses") + "Status"
	case strings.HasSuffix(name, "statuses") && len(name) > len("statuses"):
		return strings.TrimSuffix(name, "statuses") + "status"
	case strings.HasSuffix(name, "Status") && len(name) > len("Status"):
		return sdkSingularize(strings.TrimSuffix(name, "Status")) + "Status"
	case strings.HasSuffix(name, "status") && len(name) > len("status"):
		return sdkSingularize(strings.TrimSuffix(name, "status")) + "status"
	case name == "Status" || name == "status":
		return name
	case name == "Stats" || name == "stats":
		return name
	case strings.HasSuffix(name, "Stats") && len(name) > len("Stats"):
		return sdkSingularize(strings.TrimSuffix(name, "Stats")) + "Stats"
	case strings.HasSuffix(name, "stats") && len(name) > len("stats"):
		return sdkSingularize(strings.TrimSuffix(name, "stats")) + "stats"
	case strings.HasSuffix(name, "ies") && len(name) > 3:
		return strings.TrimSuffix(name, "ies") + "y"
	case strings.HasSuffix(name, "sses"),
		strings.HasSuffix(name, "shes"),
		strings.HasSuffix(name, "ches"),
		strings.HasSuffix(name, "xes"),
		strings.HasSuffix(name, "zes"):
		return strings.TrimSuffix(name, "es")
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss"):
		return strings.TrimSuffix(name, "s")
	default:
		return name
	}
}
