package generator

import (
	"fmt"
	"sort"
	"strings"
)

type mutabilityOverlayPathResolver struct {
	fields []mutabilityOverlayFieldNode
}

type mutabilityOverlayFieldNode struct {
	jsonName string
	kind     mutabilityOverlayFieldKind
	children []mutabilityOverlayFieldNode
}

type mutabilityOverlayFieldKind string

const (
	mutabilityOverlayFieldKindScalar mutabilityOverlayFieldKind = "scalar"
	mutabilityOverlayFieldKindObject mutabilityOverlayFieldKind = "object"
	mutabilityOverlayFieldKindList   mutabilityOverlayFieldKind = "list"
	mutabilityOverlayFieldKindMap    mutabilityOverlayFieldKind = "map"
)

type mutabilityOverlayPathToken struct {
	raw      string
	key      string
	listItem bool
	wildcard bool
}

// mutabilityOverlayCanonicalFieldPath captures one resolved canonical join target.
type mutabilityOverlayCanonicalFieldPath struct {
	CanonicalJoinKey string
	PathShape        string
}

// mutabilityOverlayPathResolution captures how one raw field path mapped to the generated spec surface.
type mutabilityOverlayPathResolution struct {
	FieldPath   string
	Status      string
	Candidates  []mutabilityOverlayCanonicalFieldPath
	Diagnostics []string
}

// mutabilityOverlayDocsEvidenceInput carries one docs field path plus parsed evidence.
type mutabilityOverlayDocsEvidenceInput struct {
	FieldPath     string
	EvidenceState string
	Detail        string
	RawSignal     string
}

// mutabilityOverlayDocsEvidenceComparison attaches canonical resolution details to one docs evidence row.
type mutabilityOverlayDocsEvidenceComparison struct {
	FieldPath     string
	EvidenceState string
	Detail        string
	RawSignal     string
	Resolution    mutabilityOverlayPathResolution
}

// mutabilityOverlayFieldComparisonInput compares one AST candidate row against docs evidence rows for the same resource.
type mutabilityOverlayFieldComparisonInput struct {
	ASTFieldPath string
	ASTState     string
	ForceNew     bool
	DocsEvidence []mutabilityOverlayDocsEvidenceInput
}

// mutabilityOverlayFieldComparison captures the canonical join outcome for one AST row.
type mutabilityOverlayFieldComparison struct {
	ASTFieldPath        string
	ASTResolution       mutabilityOverlayPathResolution
	TerraformFieldPath  string
	TerraformResolution mutabilityOverlayPathResolution
	JoinStatus          string
	CandidateDocs       []mutabilityOverlayDocsEvidenceComparison
	Diagnostics         []string
	Merge               mutabilityOverlayMergeResult
}

func newMutabilityOverlayPathResolver(resource ResourceModel) mutabilityOverlayPathResolver {
	helperIndex := make(map[string]TypeModel, len(resource.HelperTypes))
	for _, helper := range resource.HelperTypes {
		helperIndex[helper.Name] = helper
	}
	return mutabilityOverlayPathResolver{
		fields: buildMutabilityOverlayFieldNodes(resource.SpecFields, helperIndex, map[string]struct{}{}),
	}
}

func buildMutabilityOverlayFieldNodes(fields []FieldModel, helperIndex map[string]TypeModel, seen map[string]struct{}) []mutabilityOverlayFieldNode {
	nodes := make([]mutabilityOverlayFieldNode, 0, len(fields))
	for _, field := range fields {
		jsonName := tagJSONName(field.Tag)
		if jsonName == "" {
			jsonName = lowerCamel(field.Name)
		}
		jsonName = strings.TrimSpace(jsonName)
		if jsonName == "" {
			continue
		}

		node := mutabilityOverlayFieldNode{
			jsonName: jsonName,
			kind:     mutabilityOverlayFieldKindScalar,
		}

		trimmedType := strings.TrimSpace(field.Type)
		switch {
		case strings.HasPrefix(trimmedType, "[]"):
			node.kind = mutabilityOverlayFieldKindList
			if helper, ok := helperIndex[underlyingTypeName(trimmedType)]; ok {
				node.children = buildMutabilityOverlayHelperChildren(helper, helperIndex, seen)
			}
		case strings.HasPrefix(trimmedType, "map[string]"):
			node.kind = mutabilityOverlayFieldKindMap
		default:
			if helper, ok := helperIndex[underlyingTypeName(trimmedType)]; ok {
				node.kind = mutabilityOverlayFieldKindObject
				node.children = buildMutabilityOverlayHelperChildren(helper, helperIndex, seen)
			}
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].jsonName < nodes[j].jsonName
	})
	return nodes
}

func buildMutabilityOverlayHelperChildren(helper TypeModel, helperIndex map[string]TypeModel, seen map[string]struct{}) []mutabilityOverlayFieldNode {
	if _, ok := seen[helper.Name]; ok {
		return nil
	}

	nextSeen := make(map[string]struct{}, len(seen)+1)
	for name := range seen {
		nextSeen[name] = struct{}{}
	}
	nextSeen[helper.Name] = struct{}{}

	return buildMutabilityOverlayFieldNodes(helper.Fields, helperIndex, nextSeen)
}

func (r mutabilityOverlayPathResolver) Resolve(fieldPath string) mutabilityOverlayPathResolution {
	resolution := mutabilityOverlayPathResolution{
		FieldPath: strings.TrimSpace(fieldPath),
		Status:    mutabilityOverlayJoinUnresolved,
	}
	tokens, err := parseMutabilityOverlayPathTokens(fieldPath)
	if err != nil {
		resolution.Diagnostics = []string{err.Error()}
		return resolution
	}

	candidates, diagnostics := resolveMutabilityOverlayFieldPath(r.fields, tokens, nil)
	resolution.Candidates = dedupeMutabilityOverlayCanonicalFieldPaths(candidates)
	resolution.Diagnostics = uniqueSortedStrings(diagnostics)

	switch len(resolution.Candidates) {
	case 0:
		resolution.Status = mutabilityOverlayJoinUnresolved
	case 1:
		resolution.Status = mutabilityOverlayJoinMatched
	default:
		resolution.Status = mutabilityOverlayJoinAmbiguous
		var keys []string
		for _, candidate := range resolution.Candidates {
			keys = append(keys, candidate.CanonicalJoinKey)
		}
		resolution.Diagnostics = uniqueSortedStrings(append(
			resolution.Diagnostics,
			fmt.Sprintf("path %q matched multiple canonical join keys: %s", resolution.FieldPath, strings.Join(keys, ", ")),
		))
	}

	return resolution
}

func (r mutabilityOverlayPathResolver) Compare(input mutabilityOverlayFieldComparisonInput) mutabilityOverlayFieldComparison {
	comparison := mutabilityOverlayFieldComparison{
		ASTFieldPath:  strings.TrimSpace(input.ASTFieldPath),
		ASTResolution: r.Resolve(input.ASTFieldPath),
	}

	docs := make([]mutabilityOverlayDocsEvidenceComparison, 0, len(input.DocsEvidence))
	for _, evidence := range input.DocsEvidence {
		docs = append(docs, mutabilityOverlayDocsEvidenceComparison{
			FieldPath:     strings.TrimSpace(evidence.FieldPath),
			EvidenceState: strings.TrimSpace(evidence.EvidenceState),
			Detail:        strings.TrimSpace(evidence.Detail),
			RawSignal:     strings.TrimSpace(evidence.RawSignal),
			Resolution:    r.Resolve(evidence.FieldPath),
		})
	}

	comparison.Diagnostics = append(comparison.Diagnostics, comparison.ASTResolution.Diagnostics...)
	if comparison.ASTResolution.Status != mutabilityOverlayJoinMatched {
		comparison.JoinStatus = comparison.ASTResolution.Status
		comparison.CandidateDocs = relevantMutabilityOverlayDocsForAmbiguousAST(comparison.ASTResolution, docs)
		comparison.Merge = resolveMutabilityOverlayDecision(
			input.ASTState,
			input.ForceNew,
			mutabilityOverlayDocsStateUnknown,
			comparison.JoinStatus,
		)
		comparison.Diagnostics = uniqueSortedStrings(comparison.Diagnostics)
		return comparison
	}

	astCanonical := comparison.ASTResolution.Candidates[0]
	exactMatched := matchedMutabilityOverlayDocsByCanonicalKey(docs, astCanonical.CanonicalJoinKey)
	exactAmbiguous := ambiguousMutabilityOverlayDocsByCanonicalKey(docs, astCanonical.CanonicalJoinKey)

	switch {
	case len(exactMatched) == 1 && len(exactAmbiguous) == 0:
		selected := exactMatched[0]
		comparison.JoinStatus = mutabilityOverlayJoinMatched
		comparison.CandidateDocs = []mutabilityOverlayDocsEvidenceComparison{selected}
		comparison.TerraformFieldPath = selected.FieldPath
		comparison.TerraformResolution = selected.Resolution
		comparison.Merge = resolveMutabilityOverlayDecision(
			input.ASTState,
			input.ForceNew,
			selected.EvidenceState,
			comparison.JoinStatus,
		)
	case len(exactMatched)+len(exactAmbiguous) > 1 || len(exactAmbiguous) > 0:
		comparison.JoinStatus = mutabilityOverlayJoinAmbiguous
		comparison.CandidateDocs = append(comparison.CandidateDocs, exactMatched...)
		comparison.CandidateDocs = append(comparison.CandidateDocs, exactAmbiguous...)
		comparison.Diagnostics = append(comparison.Diagnostics,
			fmt.Sprintf("AST canonical join key %q matched multiple docs paths", astCanonical.CanonicalJoinKey),
		)
		comparison.Merge = resolveMutabilityOverlayDecision(
			input.ASTState,
			input.ForceNew,
			mutabilityOverlayDocsStateUnknown,
			comparison.JoinStatus,
		)
	default:
		familyKey := mutabilityOverlayJoinFamilyKey(astCanonical.CanonicalJoinKey)
		relatedMatched := relatedMutabilityOverlayDocsByFamilyKey(docs, astCanonical.CanonicalJoinKey, familyKey)
		relatedAmbiguous := ambiguousMutabilityOverlayDocsByFamilyKey(docs, astCanonical.CanonicalJoinKey, familyKey)

		switch {
		case len(relatedMatched) == 1 && len(relatedAmbiguous) == 0:
			selected := relatedMatched[0]
			comparison.JoinStatus = mutabilityOverlayJoinUnresolved
			comparison.CandidateDocs = []mutabilityOverlayDocsEvidenceComparison{selected}
			comparison.TerraformFieldPath = selected.FieldPath
			comparison.TerraformResolution = selected.Resolution
			comparison.Diagnostics = append(comparison.Diagnostics,
				fmt.Sprintf(
					"AST canonical join key %q had no exact docs match; related docs key %q stays unresolved",
					astCanonical.CanonicalJoinKey,
					selected.Resolution.Candidates[0].CanonicalJoinKey,
				),
			)
			comparison.Merge = resolveMutabilityOverlayDecision(
				input.ASTState,
				input.ForceNew,
				selected.EvidenceState,
				comparison.JoinStatus,
			)
		case len(relatedMatched)+len(relatedAmbiguous) > 1 || len(relatedAmbiguous) > 0:
			comparison.JoinStatus = mutabilityOverlayJoinAmbiguous
			comparison.CandidateDocs = append(comparison.CandidateDocs, relatedMatched...)
			comparison.CandidateDocs = append(comparison.CandidateDocs, relatedAmbiguous...)
			comparison.Diagnostics = append(comparison.Diagnostics,
				fmt.Sprintf("AST canonical join key %q had multiple related docs paths", astCanonical.CanonicalJoinKey),
			)
			comparison.Merge = resolveMutabilityOverlayDecision(
				input.ASTState,
				input.ForceNew,
				mutabilityOverlayDocsStateUnknown,
				comparison.JoinStatus,
			)
		default:
			comparison.JoinStatus = mutabilityOverlayJoinMatched
			comparison.TerraformFieldPath = comparison.ASTFieldPath
			comparison.TerraformResolution = comparison.ASTResolution
			comparison.Merge = resolveMutabilityOverlayDecision(
				input.ASTState,
				input.ForceNew,
				mutabilityOverlayDocsStateNotDocumented,
				comparison.JoinStatus,
			)
		}
	}

	for _, doc := range comparison.CandidateDocs {
		comparison.Diagnostics = append(comparison.Diagnostics, doc.Resolution.Diagnostics...)
	}
	comparison.Diagnostics = uniqueSortedStrings(comparison.Diagnostics)
	return comparison
}

func parseMutabilityOverlayPathTokens(fieldPath string) ([]mutabilityOverlayPathToken, error) {
	fieldPath = strings.TrimSpace(fieldPath)
	if fieldPath == "" {
		return nil, fmt.Errorf("field path must not be empty")
	}

	parts := strings.Split(fieldPath, ".")
	tokens := make([]mutabilityOverlayPathToken, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("field path %q contains a blank segment", fieldPath)
		}
		if part == "*" {
			tokens = append(tokens, mutabilityOverlayPathToken{
				raw:      part,
				wildcard: true,
			})
			continue
		}

		listItem := false
		if strings.HasSuffix(part, "[]") {
			listItem = true
			part = strings.TrimSuffix(part, "[]")
		}
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("field path %q contains an empty list segment", fieldPath)
		}

		token := mutabilityOverlayPathToken{
			raw:      part,
			key:      mutabilityOverlayPathSegmentKey(part),
			listItem: listItem,
		}
		if token.key == "" {
			return nil, fmt.Errorf("field path %q contains an invalid segment %q", fieldPath, part)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func resolveMutabilityOverlayFieldPath(fields []mutabilityOverlayFieldNode, tokens []mutabilityOverlayPathToken, prefix []string) ([]mutabilityOverlayCanonicalFieldPath, []string) {
	if len(tokens) == 0 {
		return nil, []string{"no field-path tokens were provided"}
	}
	if tokens[0].wildcard {
		location := "<root>"
		if len(prefix) > 0 {
			location = strings.Join(prefix, ".")
		}
		return nil, []string{fmt.Sprintf("unexpected wildcard under %s", location)}
	}

	type pathMatch struct {
		field    mutabilityOverlayFieldNode
		singular bool
	}

	var exactMatches []pathMatch
	var aliasMatches []pathMatch
	for _, field := range fields {
		exact, singular := field.matches(tokens[0])
		if !exact && !singular {
			continue
		}
		if singular {
			aliasMatches = append(aliasMatches, pathMatch{field: field, singular: true})
			continue
		}
		exactMatches = append(exactMatches, pathMatch{field: field})
	}

	selectedMatches := exactMatches
	if len(selectedMatches) == 0 {
		selectedMatches = aliasMatches
	}
	if len(selectedMatches) != 0 {
		var (
			candidates  []mutabilityOverlayCanonicalFieldPath
			diagnostics []string
		)
		for _, match := range selectedMatches {
			nextCandidates, nextDiagnostics := resolveMutabilityOverlayFieldNode(match.field, tokens[0], tokens[1:], prefix, match.singular)
			candidates = append(candidates, nextCandidates...)
			diagnostics = append(diagnostics, nextDiagnostics...)
		}
		return candidates, diagnostics
	}

	location := "<root>"
	if len(prefix) > 0 {
		location = strings.Join(prefix, ".")
	}
	return nil, []string{fmt.Sprintf("segment %q did not match any generated field under %s", tokens[0].raw, location)}
}

func resolveMutabilityOverlayFieldNode(field mutabilityOverlayFieldNode, token mutabilityOverlayPathToken, remaining []mutabilityOverlayPathToken, prefix []string, singularAlias bool) ([]mutabilityOverlayCanonicalFieldPath, []string) {
	current := appendClonedStrings(prefix, field.jsonName)
	switch field.kind {
	case mutabilityOverlayFieldKindScalar:
		if token.listItem {
			return nil, []string{fmt.Sprintf("field %q is not a list", field.jsonName)}
		}
		if len(remaining) > 0 {
			return nil, []string{fmt.Sprintf("field %q does not expose nested path %q", field.jsonName, remaining[0].raw)}
		}
		return []mutabilityOverlayCanonicalFieldPath{newMutabilityOverlayCanonicalFieldPath(current, field.kind)}, nil
	case mutabilityOverlayFieldKindObject:
		if token.listItem {
			return nil, []string{fmt.Sprintf("field %q is not a list", field.jsonName)}
		}
		if len(remaining) == 0 {
			return []mutabilityOverlayCanonicalFieldPath{newMutabilityOverlayCanonicalFieldPath(current, field.kind)}, nil
		}
		return resolveMutabilityOverlayFieldPath(field.children, remaining, current)
	case mutabilityOverlayFieldKindList:
		if len(remaining) == 0 {
			if token.listItem || singularAlias {
				current[len(current)-1] = field.jsonName + "[]"
				return []mutabilityOverlayCanonicalFieldPath{newMutabilityOverlayCanonicalFieldPath(current, field.kind)}, nil
			}
			return []mutabilityOverlayCanonicalFieldPath{newMutabilityOverlayCanonicalFieldPath(current, field.kind)}, nil
		}

		current[len(current)-1] = field.jsonName + "[]"
		if len(field.children) == 0 {
			return nil, []string{fmt.Sprintf("list field %q does not expose nested path %q", field.jsonName, remaining[0].raw)}
		}
		return resolveMutabilityOverlayFieldPath(field.children, remaining, current)
	case mutabilityOverlayFieldKindMap:
		if token.listItem {
			return nil, []string{fmt.Sprintf("field %q is not a list", field.jsonName)}
		}
		if len(remaining) == 0 {
			return []mutabilityOverlayCanonicalFieldPath{newMutabilityOverlayCanonicalFieldPath(current, field.kind)}, nil
		}
		if len(remaining) == 1 && remaining[0].wildcard {
			current = append(current, "*")
			return []mutabilityOverlayCanonicalFieldPath{newMutabilityOverlayCanonicalFieldPath(current, field.kind)}, nil
		}
		return nil, []string{fmt.Sprintf("map field %q only supports a terminal .* path", field.jsonName)}
	default:
		return nil, []string{fmt.Sprintf("field %q has unsupported kind %q", field.jsonName, field.kind)}
	}
}

func (field mutabilityOverlayFieldNode) matches(token mutabilityOverlayPathToken) (bool, bool) {
	if token.wildcard {
		return false, false
	}

	tokenKey := token.key
	if tokenKey == "" {
		return false, false
	}

	fieldKey := mutabilityOverlayPathSegmentKey(field.jsonName)
	if tokenKey == fieldKey {
		if token.listItem && field.kind != mutabilityOverlayFieldKindList {
			return false, false
		}
		return true, false
	}

	if field.kind != mutabilityOverlayFieldKindList {
		return false, false
	}
	if tokenKey == mutabilityOverlayPathSegmentKey(singularize(field.jsonName)) {
		return true, true
	}
	return false, false
}

func newMutabilityOverlayCanonicalFieldPath(segments []string, terminalKind mutabilityOverlayFieldKind) mutabilityOverlayCanonicalFieldPath {
	return mutabilityOverlayCanonicalFieldPath{
		CanonicalJoinKey: strings.Join(segments, "."),
		PathShape:        inferMutabilityOverlayPathShape(segments, terminalKind),
	}
}

func inferMutabilityOverlayPathShape(segments []string, terminalKind mutabilityOverlayFieldKind) string {
	for _, segment := range segments {
		if segment == "*" {
			return mutabilityOverlayPathShapeMapEntry
		}
		if strings.HasSuffix(segment, "[]") {
			return mutabilityOverlayPathShapeListItem
		}
	}
	if len(segments) > 1 || terminalKind == mutabilityOverlayFieldKindObject {
		return mutabilityOverlayPathShapeObject
	}
	return mutabilityOverlayPathShapeScalar
}

func dedupeMutabilityOverlayCanonicalFieldPaths(paths []mutabilityOverlayCanonicalFieldPath) []mutabilityOverlayCanonicalFieldPath {
	if len(paths) == 0 {
		return nil
	}

	type key struct {
		joinKey   string
		pathShape string
	}
	seen := make(map[key]struct{}, len(paths))
	out := make([]mutabilityOverlayCanonicalFieldPath, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path.CanonicalJoinKey) == "" {
			continue
		}
		k := key{joinKey: path.CanonicalJoinKey, pathShape: path.PathShape}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, path)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CanonicalJoinKey != out[j].CanonicalJoinKey {
			return out[i].CanonicalJoinKey < out[j].CanonicalJoinKey
		}
		return out[i].PathShape < out[j].PathShape
	})
	return out
}

func relevantMutabilityOverlayDocsForAmbiguousAST(astResolution mutabilityOverlayPathResolution, docs []mutabilityOverlayDocsEvidenceComparison) []mutabilityOverlayDocsEvidenceComparison {
	if len(astResolution.Candidates) == 0 {
		return nil
	}

	var relevant []mutabilityOverlayDocsEvidenceComparison
	for _, doc := range docs {
		for _, candidate := range doc.Resolution.Candidates {
			for _, astCandidate := range astResolution.Candidates {
				if candidate.CanonicalJoinKey == astCandidate.CanonicalJoinKey || mutabilityOverlayJoinFamilyKey(candidate.CanonicalJoinKey) == mutabilityOverlayJoinFamilyKey(astCandidate.CanonicalJoinKey) {
					relevant = append(relevant, doc)
					goto nextDoc
				}
			}
		}
	nextDoc:
	}
	return relevant
}

func matchedMutabilityOverlayDocsByCanonicalKey(docs []mutabilityOverlayDocsEvidenceComparison, joinKey string) []mutabilityOverlayDocsEvidenceComparison {
	var matched []mutabilityOverlayDocsEvidenceComparison
	for _, doc := range docs {
		if doc.Resolution.Status != mutabilityOverlayJoinMatched || len(doc.Resolution.Candidates) != 1 {
			continue
		}
		if doc.Resolution.Candidates[0].CanonicalJoinKey == joinKey {
			matched = append(matched, doc)
		}
	}
	return matched
}

func ambiguousMutabilityOverlayDocsByCanonicalKey(docs []mutabilityOverlayDocsEvidenceComparison, joinKey string) []mutabilityOverlayDocsEvidenceComparison {
	var ambiguous []mutabilityOverlayDocsEvidenceComparison
	for _, doc := range docs {
		if doc.Resolution.Status != mutabilityOverlayJoinAmbiguous {
			continue
		}
		for _, candidate := range doc.Resolution.Candidates {
			if candidate.CanonicalJoinKey == joinKey {
				ambiguous = append(ambiguous, doc)
				break
			}
		}
	}
	return ambiguous
}

func relatedMutabilityOverlayDocsByFamilyKey(docs []mutabilityOverlayDocsEvidenceComparison, joinKey string, familyKey string) []mutabilityOverlayDocsEvidenceComparison {
	var related []mutabilityOverlayDocsEvidenceComparison
	for _, doc := range docs {
		if doc.Resolution.Status != mutabilityOverlayJoinMatched || len(doc.Resolution.Candidates) != 1 {
			continue
		}
		candidate := doc.Resolution.Candidates[0]
		if candidate.CanonicalJoinKey == joinKey {
			continue
		}
		if mutabilityOverlayJoinFamilyKey(candidate.CanonicalJoinKey) == familyKey {
			related = append(related, doc)
		}
	}
	return related
}

func ambiguousMutabilityOverlayDocsByFamilyKey(docs []mutabilityOverlayDocsEvidenceComparison, joinKey string, familyKey string) []mutabilityOverlayDocsEvidenceComparison {
	var ambiguous []mutabilityOverlayDocsEvidenceComparison
	for _, doc := range docs {
		if doc.Resolution.Status != mutabilityOverlayJoinAmbiguous {
			continue
		}
		for _, candidate := range doc.Resolution.Candidates {
			if candidate.CanonicalJoinKey == joinKey {
				continue
			}
			if mutabilityOverlayJoinFamilyKey(candidate.CanonicalJoinKey) == familyKey {
				ambiguous = append(ambiguous, doc)
				break
			}
		}
	}
	return ambiguous
}

func mutabilityOverlayJoinFamilyKey(joinKey string) string {
	joinKey = strings.ReplaceAll(joinKey, "[]", "")
	joinKey = strings.ReplaceAll(joinKey, ".*", "")
	return joinKey
}

func mutabilityOverlayPathSegmentKey(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "[]")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ToLower(value)
	return value
}

func appendClonedStrings(values []string, value string) []string {
	out := append([]string(nil), values...)
	out = append(out, value)
	return out
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
