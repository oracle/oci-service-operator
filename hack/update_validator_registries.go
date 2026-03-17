package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type specTarget struct {
	Group    string
	Spec     string
	Name     string
	SDKTypes []string
}

type sdkTarget struct {
	Group string
	Type  string
}

var (
	reSpecType  = regexp.MustCompile(`type\s+([A-Za-z0-9]+)Spec\s+struct\s*\{`)
	reSDKStruct = regexp.MustCompile(`(?m)^type\s+([A-Za-z0-9]+)\s+struct\b`)

	reAPITargetBlock = regexp.MustCompile(`(?s)\{\s*Name:\s*"([^"]+)",\s*SpecType:\s*reflect\.TypeOf\(([a-z0-9]+)v1beta1\.([A-Za-z0-9]+)Spec\{\}\),\s*SDKStructs:\s*\[]string\{\s*(.*?)\s*\},\s*\},`)
	reSDKRef         = regexp.MustCompile(`"([a-z0-9]+\.[A-Za-z0-9]+)"`)
	reSDKTarget      = regexp.MustCompile(`newTarget\("([a-z0-9]+)",\s*"([A-Za-z0-9]+)"`)
)

func main() {
	write := flag.Bool("write", false, "write changes to registry files")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		die(err)
	}

	apispecPath := filepath.Join(root, "internal", "validator", "apispec", "registry.go")
	sdkPath := filepath.Join(root, "internal", "validator", "sdk", "registry.go")

	existingAPI, err := parseExistingAPITargets(apispecPath)
	if err != nil {
		die(err)
	}
	existingSDK, err := parseExistingSDKTargets(sdkPath)
	if err != nil {
		die(err)
	}

	apiSpecs, err := scanAPISpecs(root)
	if err != nil {
		die(err)
	}

	targets := buildTargets(root, apiSpecs, existingAPI)
	apiOut, err := renderAPIRegistry(targets)
	if err != nil {
		die(err)
	}

	sdkTargets := buildSDKTargets(targets, existingSDK)
	sdkOut, err := renderSDKRegistry(sdkTargets)
	if err != nil {
		die(err)
	}

	if !*write {
		reportDiff(apispecPath, apiOut)
		reportDiff(sdkPath, sdkOut)
		fmt.Println("Run with --write to apply changes.")
		return
	}

	if err := os.WriteFile(apispecPath, apiOut, 0o644); err != nil {
		die(err)
	}
	if err := os.WriteFile(sdkPath, sdkOut, 0o644); err != nil {
		die(err)
	}

	fmt.Printf("Updated %s\n", rel(root, apispecPath))
	fmt.Printf("Updated %s\n", rel(root, sdkPath))
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}

func parseExistingAPITargets(path string) (map[string]specTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string]specTarget)
	for _, block := range reAPITargetBlock.FindAllSubmatch(data, -1) {
		name := string(block[1])
		group := string(block[2])
		spec := string(block[3])
		sdkRefsRaw := block[4]
		refs := make([]string, 0)
		for _, ref := range reSDKRef.FindAllSubmatch(sdkRefsRaw, -1) {
			refs = append(refs, string(ref[1]))
		}
		key := group + "." + spec
		m[key] = specTarget{Group: group, Spec: spec, Name: name, SDKTypes: refs}
	}
	return m, nil
}

func parseExistingSDKTargets(path string) ([]sdkTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := make([]sdkTarget, 0)
	for _, m := range reSDKTarget.FindAllSubmatch(data, -1) {
		out = append(out, sdkTarget{Group: string(m[1]), Type: string(m[2])})
	}
	return out, nil
}

func scanAPISpecs(root string) (map[string][]string, error) {
	apiRoot := filepath.Join(root, "api")
	out := make(map[string][]string)
	err := filepath.WalkDir(apiRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_types.go") {
			return nil
		}
		relPath, err := filepath.Rel(apiRoot, path)
		if err != nil {
			return err
		}
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) != 3 || parts[1] != "v1beta1" {
			return nil
		}
		group := parts[0]
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range reSpecType.FindAllSubmatch(content, -1) {
			out[group] = append(out[group], string(m[1]))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for g := range out {
		out[g] = uniqueSorted(out[g])
	}
	return out, nil
}

func buildTargets(root string, apiSpecs map[string][]string, existing map[string]specTarget) []specTarget {
	groups := make([]string, 0, len(apiSpecs))
	for g := range apiSpecs {
		sdkDir := filepath.Join(root, "vendor", "github.com", "oracle", "oci-go-sdk", "v65", g)
		if stat, err := os.Stat(sdkDir); err == nil && stat.IsDir() {
			groups = append(groups, g)
		}
	}
	sort.Strings(groups)

	out := make([]specTarget, 0)
	for _, g := range groups {
		sdkStructs := scanSDKStructNames(filepath.Join(root, "vendor", "github.com", "oracle", "oci-go-sdk", "v65", g))
		for _, spec := range apiSpecs[g] {
			key := g + "." + spec
			existingTarget, hasExisting := existing[key]
			candidates := deriveSDKTypes(spec, sdkStructs)
			if hasExisting {
				for _, ref := range existingTarget.SDKTypes {
					parts := strings.Split(ref, ".")
					if len(parts) == 2 && parts[0] == g {
						candidates = append(candidates, parts[1])
					}
				}
			}
			candidates = uniqueByOrder(candidates)
			sort.SliceStable(candidates, func(i, j int) bool {
				ai, aj := sdkTypeOrder(candidates[i]), sdkTypeOrder(candidates[j])
				if ai != aj {
					return ai < aj
				}
				return candidates[i] < candidates[j]
			})

			if len(candidates) == 0 {
				continue
			}

			name := makeTargetName(g, spec)
			if hasExisting && strings.TrimSpace(existingTarget.Name) != "" {
				name = existingTarget.Name
			}

			sdkRefs := make([]string, 0, len(candidates))
			for _, c := range candidates {
				sdkRefs = append(sdkRefs, g+"."+c)
			}
			out = append(out, specTarget{Group: g, Spec: spec, Name: name, SDKTypes: sdkRefs})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		gi, gj := groupOrder(out[i].Group), groupOrder(out[j].Group)
		if gi != gj {
			return gi < gj
		}
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})

	return out
}

func buildSDKTargets(targets []specTarget, existing []sdkTarget) []sdkTarget {
	set := make(map[string]sdkTarget)
	for _, t := range targets {
		for _, ref := range t.SDKTypes {
			parts := strings.Split(ref, ".")
			if len(parts) != 2 {
				continue
			}
			k := ref
			set[k] = sdkTarget{Group: parts[0], Type: parts[1]}
		}
	}
	for _, e := range existing {
		k := e.Group + "." + e.Type
		if _, ok := set[k]; !ok {
			set[k] = e
		}
	}
	out := make([]sdkTarget, 0, len(set))
	for _, v := range set {
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool {
		gi, gj := groupOrder(out[i].Group), groupOrder(out[j].Group)
		if gi != gj {
			return gi < gj
		}
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		ai, aj := sdkTypeOrder(out[i].Type), sdkTypeOrder(out[j].Type)
		if ai != aj {
			return ai < aj
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func scanSDKStructNames(dir string) map[string]bool {
	out := make(map[string]bool)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range reSDKStruct.FindAllSubmatch(data, -1) {
			name := string(m[1])
			if name == "" {
				continue
			}
			if name[0] < 'A' || name[0] > 'Z' {
				continue
			}
			out[name] = true
		}
		return nil
	})
	return out
}

func deriveSDKTypes(spec string, structs map[string]bool) []string {
	variants := specVariants(spec)
	out := make([]string, 0)
	for _, v := range variants {
		addIf(&out, structs, "Create"+v+"Details")
		addIf(&out, structs, "Update"+v+"Details")
		addIf(&out, structs, v)
		addIf(&out, structs, v+"Summary")
		addIf(&out, structs, v+"VersionSummary")
		if strings.HasSuffix(v, "Bundle") {
			addIf(&out, structs, v+"PublicOnly")
		}
	}

	if len(out) == 0 && strings.HasSuffix(spec, "ByName") {
		base := strings.TrimSuffix(spec, "ByName")
		for _, v := range specVariants(base) {
			addIf(&out, structs, v)
			addIf(&out, structs, v+"Summary")
		}
	}

	return uniqueByOrder(out)
}

func addIf(out *[]string, set map[string]bool, name string) {
	if set[name] {
		*out = append(*out, name)
	}
}

func specVariants(spec string) []string {
	variants := []string{spec}
	repl := []struct{ old, new string }{
		{"IPSec", "IpSec"},
		{"CPE", "Cpe"},
		{"VCN", "Vcn"},
		{"VLAN", "Vlan"},
		{"VNIC", "Vnic"},
		{"NAT", "Nat"},
		{"DRG", "Drg"},
		{"KMS", "Kms"},
		{"OAuth", "OAuth2"},
	}
	v := spec
	for _, r := range repl {
		v = strings.ReplaceAll(v, r.old, r.new)
	}
	if v != spec {
		variants = append(variants, v)
	}
	return uniqueByOrder(variants)
}

func uniqueByOrder(in []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func uniqueSorted(in []string) []string {
	in = uniqueByOrder(in)
	sort.Strings(in)
	return in
}

func sdkTypeOrder(t string) int {
	switch {
	case strings.HasPrefix(t, "Create") && strings.HasSuffix(t, "Details"):
		return 0
	case strings.HasPrefix(t, "Update") && strings.HasSuffix(t, "Details"):
		return 1
	case strings.HasSuffix(t, "Details"):
		return 2
	case strings.HasSuffix(t, "VersionSummary"):
		return 4
	case strings.HasSuffix(t, "Summary"):
		return 5
	default:
		return 3
	}
}

func renderAPIRegistry(targets []specTarget) ([]byte, error) {
	groups := make(map[string]bool)
	for _, t := range targets {
		groups[t.Group] = true
	}
	apiGroups := make([]string, 0, len(groups))
	for g := range groups {
		apiGroups = append(apiGroups, g)
	}
	sort.SliceStable(apiGroups, func(i, j int) bool {
		gi, gj := groupOrder(apiGroups[i]), groupOrder(apiGroups[j])
		if gi != gj {
			return gi < gj
		}
		return apiGroups[i] < apiGroups[j]
	})

	var b strings.Builder
	b.WriteString("package apispec\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"reflect\"\n\n")
	for _, g := range apiGroups {
		fmt.Fprintf(&b, "\t%sv1beta1 \"github.com/oracle/oci-service-operator/api/%s/v1beta1\"\n", g, g)
	}
	b.WriteString(")\n\n")
	b.WriteString("type Target struct {\n\tName       string\n\tSpecType   reflect.Type\n\tSDKStructs []string\n}\n\n")
	b.WriteString("var targets = []Target{\n")
	for _, t := range targets {
		b.WriteString("\t{\n")
		fmt.Fprintf(&b, "\t\tName:     %q,\n", t.Name)
		fmt.Fprintf(&b, "\t\tSpecType: reflect.TypeOf(%sv1beta1.%sSpec{}),\n", t.Group, t.Spec)
		b.WriteString("\t\tSDKStructs: []string{\n")
		for _, ref := range t.SDKTypes {
			fmt.Fprintf(&b, "\t\t\t%q,\n", ref)
		}
		b.WriteString("\t\t},\n")
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func Targets() []Target {\n")
	b.WriteString("\tresult := make([]Target, len(targets))\n")
	b.WriteString("\tcopy(result, targets)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n")

	return format.Source([]byte(b.String()))
}

func renderSDKRegistry(targets []sdkTarget) ([]byte, error) {
	groups := make(map[string]bool)
	for _, t := range targets {
		groups[t.Group] = true
	}
	groups["mysql"] = true // required for interfaceImplementations map below.

	sdkGroups := make([]string, 0, len(groups))
	for g := range groups {
		sdkGroups = append(sdkGroups, g)
	}
	sort.SliceStable(sdkGroups, func(i, j int) bool {
		gi, gj := groupOrder(sdkGroups[i]), groupOrder(sdkGroups[j])
		if gi != gj {
			return gi < gj
		}
		return sdkGroups[i] < sdkGroups[j]
	})

	byGroup := make(map[string][]string)
	for _, t := range targets {
		byGroup[t.Group] = append(byGroup[t.Group], t.Type)
	}
	for g := range byGroup {
		byGroup[g] = uniqueSorted(byGroup[g])
		sort.SliceStable(byGroup[g], func(i, j int) bool {
			ai, aj := sdkTypeOrder(byGroup[g][i]), sdkTypeOrder(byGroup[g][j])
			if ai != aj {
				return ai < aj
			}
			return byGroup[g][i] < byGroup[g][j]
		})
	}

	var b strings.Builder
	b.WriteString("package sdk\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"path\"\n")
	b.WriteString("\t\"reflect\"\n\n")
	for _, g := range sdkGroups {
		fmt.Fprintf(&b, "\t\"github.com/oracle/oci-go-sdk/v65/%s\"\n", g)
	}
	b.WriteString(")\n\n")
	b.WriteString("const (\n\tmodulePath    = \"github.com/oracle/oci-go-sdk/v65\"\n\tmoduleVersion = \"v65.61.1\"\n)\n\n")
	b.WriteString("var seedTargets = []Target{\n")
	for _, g := range sdkGroups {
		types := byGroup[g]
		if len(types) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t// %s\n", serviceComment(g))
		for _, t := range types {
			fmt.Fprintf(&b, "\tnewTarget(%q, %q, reflect.TypeOf(%s.%s{})),\n", g, t, g, t)
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n\n")

	b.WriteString("var interfaceImplementations = map[string][]reflect.Type{\n")
	b.WriteString("\tqualifiedTypeName(reflect.TypeOf((*mysql.CreateDbSystemSourceDetails)(nil)).Elem()): {\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceFromBackupDetails{}),\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceFromNoneDetails{}),\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceFromPitrDetails{}),\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceImportFromUrlDetails{}),\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("func SeedTargets() []Target {\n")
	b.WriteString("\tresult := make([]Target, len(seedTargets))\n")
	b.WriteString("\tcopy(result, seedTargets)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n\n")

	b.WriteString("func TargetByName(qualifiedName string) (Target, bool) {\n")
	b.WriteString("\tfor _, target := range seedTargets {\n")
	b.WriteString("\t\tif target.QualifiedName == qualifiedName {\n")
	b.WriteString("\t\t\treturn target, true\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Target{}, false\n")
	b.WriteString("}\n\n")

	b.WriteString("func knownInterfaceImplementations(interfaceType reflect.Type) []reflect.Type {\n")
	b.WriteString("\tknown := interfaceImplementations[qualifiedTypeName(interfaceType)]\n")
	b.WriteString("\tresult := make([]reflect.Type, len(known))\n")
	b.WriteString("\tcopy(result, known)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n\n")

	b.WriteString("func newTarget(packageName string, typeName string, typeRef reflect.Type) Target {\n")
	b.WriteString("\treturn Target{\n")
	b.WriteString("\t\tQualifiedName: packageName + \".\" + typeName,\n")
	b.WriteString("\t\tPackageName:   packageName,\n")
	b.WriteString("\t\tTypeName:      typeName,\n")
	b.WriteString("\t\tImportPath:    typeRef.PkgPath(),\n")
	b.WriteString("\t\tReflectType:   typeRef,\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	b.WriteString("func qualifiedTypeName(typeRef reflect.Type) string {\n")
	b.WriteString("\treturn path.Base(typeRef.PkgPath()) + \".\" + typeRef.Name()\n")
	b.WriteString("}\n")

	return format.Source([]byte(b.String()))
}

func reportDiff(path string, next []byte) {
	cur, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("%s: unreadable (%v)\n", path, err)
		return
	}
	if bytes.Equal(cur, next) {
		fmt.Printf("%s: up to date\n", path)
		return
	}
	fmt.Printf("%s: would change\n", path)
}

func makeTargetName(group, spec string) string {
	prefix := map[string]string{
		"database":               "",
		"mysql":                  "MySql",
		"streaming":              "",
		"queue":                  "",
		"functions":              "Functions",
		"nosql":                  "NoSQL",
		"objectstorage":          "ObjectStorage",
		"ons":                    "Notification",
		"logging":                "Logging",
		"psql":                   "PSQL",
		"events":                 "Events",
		"monitoring":             "Monitoring",
		"dns":                    "DNS",
		"loadbalancer":           "LoadBalancer",
		"networkloadbalancer":    "NetworkLoadBalancer",
		"artifacts":              "Artifacts",
		"certificates":           "Certificates",
		"certificatesmanagement": "CertificatesManagement",
		"containerengine":        "ContainerEngine",
		"identity":               "Identity",
		"keymanagement":          "KeyManagement",
		"limits":                 "Limits",
		"secrets":                "Secrets",
		"vault":                  "Vault",
		"core":                   "Core",
	}
	p, ok := prefix[group]
	if !ok {
		p = pascal(group)
	}
	if p == "" {
		return spec
	}
	return p + spec
}

func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' })
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func serviceComment(group string) string {
	labels := map[string]string{
		"database":               "Autonomous Database",
		"mysql":                  "MySQL DB System",
		"streaming":              "Streaming",
		"queue":                  "Queue",
		"functions":              "Functions",
		"nosql":                  "NoSQL",
		"objectstorage":          "Object Storage",
		"ons":                    "Notifications (ONS)",
		"logging":                "Logging",
		"psql":                   "PostgreSQL",
		"events":                 "Events",
		"monitoring":             "Monitoring",
		"dns":                    "DNS",
		"loadbalancer":           "Load Balancer",
		"networkloadbalancer":    "Network Load Balancer",
		"artifacts":              "Artifacts",
		"certificates":           "Certificates",
		"certificatesmanagement": "Certificates Management",
		"containerengine":        "Container Engine",
		"identity":               "Identity",
		"keymanagement":          "Key Management",
		"limits":                 "Limits",
		"secrets":                "Secrets",
		"vault":                  "Vault",
		"core":                   "Core VCN",
	}
	if v, ok := labels[group]; ok {
		return v + " CRD support"
	}
	return pascal(group) + " CRD support"
}

func groupOrder(group string) int {
	order := []string{
		"database", "mysql", "streaming", "queue", "functions", "nosql", "objectstorage", "ons", "logging", "psql", "events", "monitoring", "dns", "loadbalancer", "networkloadbalancer", "artifacts", "certificates", "certificatesmanagement", "containerengine", "identity", "keymanagement", "limits", "secrets", "vault", "core",
	}
	for i, g := range order {
		if g == group {
			return i
		}
	}
	return len(order) + 100
}
