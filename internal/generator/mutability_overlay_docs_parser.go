package generator

import (
	"fmt"
	"sort"
	"strings"

	xhtml "golang.org/x/net/html"
)

const (
	mutabilityOverlayDocsSectionArgumentReference = "Argument Reference"

	mutabilityOverlayDocsParseReasonInvalidHTML              = "invalidHTML"
	mutabilityOverlayDocsParseReasonMissingArgumentReference = "missingArgumentReference"
	mutabilityOverlayDocsParseReasonNoFieldEntries           = "noFieldEntries"
	mutabilityOverlayDocsParseReasonPartialParse             = "partialParse"

	mutabilityOverlayDocsParseDiagnosticSeverityWarning = "warning"
	mutabilityOverlayDocsParseDiagnosticSeverityError   = "error"

	mutabilityOverlayDocsParseDiagnosticUnsupportedSectionNode = "unsupportedSectionNode"
	mutabilityOverlayDocsParseDiagnosticUnsupportedFieldEntry  = "unsupportedFieldEntry"
	mutabilityOverlayDocsParseDiagnosticDuplicateFieldPath     = "duplicateFieldPath"
)

// mutabilityOverlayParsedDocsEvidence captures one raw Terraform docs field row
// before canonical key joins or AST-primary merge logic are applied.
type mutabilityOverlayParsedDocsEvidence struct {
	FieldPath     string                                `json:"fieldPath"`
	EvidenceState string                                `json:"evidenceState"`
	Detail        string                                `json:"detail,omitempty"`
	RawSignal     string                                `json:"rawSignal,omitempty"`
	Provenance    mutabilityOverlayDocsEvidenceLocation `json:"provenance"`
}

type mutabilityOverlayDocsEvidenceLocation struct {
	RegistryURL   string `json:"registryURL"`
	SectionTitle  string `json:"sectionTitle"`
	SectionAnchor string `json:"sectionAnchor,omitempty"`
	EvidenceText  string `json:"evidenceText"`
}

// mutabilityOverlayDocsParserDiagnostic reports one actionable parser warning or
// error without hiding already-parsed field rows.
type mutabilityOverlayDocsParserDiagnostic struct {
	Severity      string `json:"severity"`
	Reason        string `json:"reason"`
	FieldPath     string `json:"fieldPath,omitempty"`
	SectionAnchor string `json:"sectionAnchor,omitempty"`
	Detail        string `json:"detail"`
	EvidenceText  string `json:"evidenceText,omitempty"`
}

// mutabilityOverlayDocsParseResult is the raw parser output consumed by later
// join and merge stages.
type mutabilityOverlayDocsParseResult struct {
	RegistryURL   string                                  `json:"registryURL"`
	SectionTitle  string                                  `json:"sectionTitle"`
	SectionAnchor string                                  `json:"sectionAnchor,omitempty"`
	Fields        []mutabilityOverlayParsedDocsEvidence   `json:"fields"`
	Diagnostics   []mutabilityOverlayDocsParserDiagnostic `json:"diagnostics,omitempty"`
}

// mutabilityOverlayDocsParseError surfaces fatal or partial parse failures with
// resource identity and collected diagnostics.
type mutabilityOverlayDocsParseError struct {
	Reason           string
	Service          string
	Kind             string
	FormalSlug       string
	ProviderResource string
	RegistryURL      string
	Detail           string
	Diagnostics      []mutabilityOverlayDocsParserDiagnostic
}

func (e *mutabilityOverlayDocsParseError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "mutability overlay docs parse failed for service %q kind %q", e.Service, e.Kind)
	if strings.TrimSpace(e.FormalSlug) != "" {
		fmt.Fprintf(&b, " formalSpec %q", e.FormalSlug)
	}
	if strings.TrimSpace(e.ProviderResource) != "" {
		fmt.Fprintf(&b, " providerResource=%q", e.ProviderResource)
	}
	if strings.TrimSpace(e.RegistryURL) != "" {
		fmt.Fprintf(&b, " url=%q", e.RegistryURL)
	}
	fmt.Fprintf(&b, ": %s", e.Reason)
	if strings.TrimSpace(e.Detail) != "" {
		fmt.Fprintf(&b, " (%s)", e.Detail)
	}
	if len(e.Diagnostics) != 0 {
		fmt.Fprintf(&b, " diagnostics=%d", len(e.Diagnostics))
	}
	return b.String()
}

// parseMutabilityOverlayDocsArgumentReference extracts raw field-level update
// evidence from one Terraform Registry HTML input without performing AST/docs
// merge or canonical join resolution.
func parseMutabilityOverlayDocsArgumentReference(input mutabilityOverlayDocsInput) (mutabilityOverlayDocsParseResult, error) {
	result := mutabilityOverlayDocsParseResult{
		RegistryURL:  strings.TrimSpace(input.Metadata.RegistryURL),
		SectionTitle: mutabilityOverlayDocsSectionArgumentReference,
	}

	doc, err := xhtml.Parse(strings.NewReader(input.Body))
	if err != nil {
		return result, newMutabilityOverlayDocsParseError(
			input,
			mutabilityOverlayDocsParseReasonInvalidHTML,
			fmt.Sprintf("parse HTML: %v", err),
			nil,
		)
	}

	heading := findMutabilityOverlayDocsSectionHeading(doc, mutabilityOverlayDocsSectionArgumentReference)
	if heading == nil {
		return result, newMutabilityOverlayDocsParseError(
			input,
			mutabilityOverlayDocsParseReasonMissingArgumentReference,
			"the HTML body did not contain an Argument Reference heading",
			nil,
		)
	}

	result.SectionAnchor = mutabilityOverlayDocsSectionAnchor(heading)
	for _, node := range mutabilityOverlayDocsSectionNodes(heading) {
		parseMutabilityOverlayDocsSectionNode(node, nil, &result)
	}
	annotateMutabilityOverlayDocsDuplicateFieldPaths(&result)

	if len(result.Fields) == 0 {
		reason := mutabilityOverlayDocsParseReasonNoFieldEntries
		detail := "Argument Reference section did not yield any field entries"
		if countMutabilityOverlayDocsErrorDiagnostics(result.Diagnostics) != 0 {
			reason = mutabilityOverlayDocsParseReasonPartialParse
			detail = fmt.Sprintf(
				"Argument Reference section recorded %d parser errors before any field rows were emitted",
				countMutabilityOverlayDocsErrorDiagnostics(result.Diagnostics),
			)
		}
		return result, newMutabilityOverlayDocsParseError(input, reason, detail, result.Diagnostics)
	}

	if countMutabilityOverlayDocsErrorDiagnostics(result.Diagnostics) != 0 {
		return result, newMutabilityOverlayDocsParseError(
			input,
			mutabilityOverlayDocsParseReasonPartialParse,
			fmt.Sprintf(
				"Argument Reference section emitted %d field rows but also recorded %d parser errors",
				len(result.Fields),
				countMutabilityOverlayDocsErrorDiagnostics(result.Diagnostics),
			),
			result.Diagnostics,
		)
	}

	return result, nil
}

func (r mutabilityOverlayDocsParseResult) EvidenceInputs() []mutabilityOverlayDocsEvidenceInput {
	out := make([]mutabilityOverlayDocsEvidenceInput, 0, len(r.Fields))
	for _, field := range r.Fields {
		out = append(out, mutabilityOverlayDocsEvidenceInput{
			FieldPath:     field.FieldPath,
			EvidenceState: field.EvidenceState,
			Detail:        field.Detail,
			RawSignal:     field.RawSignal,
		})
	}
	return out
}

func newMutabilityOverlayDocsParseError(
	input mutabilityOverlayDocsInput,
	reason string,
	detail string,
	diagnostics []mutabilityOverlayDocsParserDiagnostic,
) error {
	return &mutabilityOverlayDocsParseError{
		Reason:           reason,
		Service:          strings.TrimSpace(input.Metadata.Service),
		Kind:             strings.TrimSpace(input.Metadata.Kind),
		FormalSlug:       strings.TrimSpace(input.Metadata.FormalSlug),
		ProviderResource: strings.TrimSpace(input.Metadata.ProviderResource),
		RegistryURL:      strings.TrimSpace(input.Metadata.RegistryURL),
		Detail:           strings.TrimSpace(detail),
		Diagnostics:      append([]mutabilityOverlayDocsParserDiagnostic(nil), diagnostics...),
	}
}

func findMutabilityOverlayDocsSectionHeading(root *xhtml.Node, title string) *xhtml.Node {
	normalizedTitle := normalizeMutabilityOverlayDocsText(title)
	var found *xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if found != nil || node == nil {
			return
		}
		if node.Type == xhtml.ElementNode && isMutabilityOverlayDocsHeading(node) {
			if normalizeMutabilityOverlayDocsText(mutabilityOverlayDocsNodeText(node)) == normalizedTitle {
				found = node
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return found
}

func mutabilityOverlayDocsSectionNodes(heading *xhtml.Node) []*xhtml.Node {
	if heading == nil || heading.Parent == nil {
		return nil
	}

	level := mutabilityOverlayDocsHeadingLevel(heading.Data)
	var nodes []*xhtml.Node
	for node := heading.NextSibling; node != nil; node = node.NextSibling {
		if node.Type == xhtml.ElementNode && isMutabilityOverlayDocsHeading(node) {
			if mutabilityOverlayDocsHeadingLevel(node.Data) <= level {
				break
			}
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func mutabilityOverlayDocsSectionAnchor(node *xhtml.Node) string {
	if node == nil {
		return ""
	}

	if id := strings.TrimSpace(mutabilityOverlayDocsNodeAttr(node, "id")); id != "" {
		return id
	}
	if name := strings.TrimSpace(mutabilityOverlayDocsNodeAttr(node, "name")); name != "" {
		return name
	}
	if href := strings.TrimSpace(mutabilityOverlayDocsNodeAttr(node, "href")); strings.HasPrefix(href, "#") {
		return strings.TrimPrefix(href, "#")
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if anchor := mutabilityOverlayDocsSectionAnchor(child); anchor != "" {
			return anchor
		}
	}
	return ""
}

func parseMutabilityOverlayDocsSectionNode(node *xhtml.Node, prefix []string, result *mutabilityOverlayDocsParseResult) {
	if node == nil || result == nil {
		return
	}

	if node.Type != xhtml.ElementNode {
		return
	}

	switch {
	case isMutabilityOverlayDocsList(node):
		parseMutabilityOverlayDocsListNode(node, prefix, result)
	case strings.EqualFold(node.Data, "table"):
		appendMutabilityOverlayDocsDiagnostic(result, mutabilityOverlayDocsParserDiagnostic{
			Severity:      mutabilityOverlayDocsParseDiagnosticSeverityError,
			Reason:        mutabilityOverlayDocsParseDiagnosticUnsupportedSectionNode,
			FieldPath:     strings.Join(prefix, "."),
			SectionAnchor: result.SectionAnchor,
			Detail:        "Argument Reference contains an unsupported <table> layout that may hide field entries",
			EvidenceText:  normalizeMutabilityOverlayDocsText(mutabilityOverlayDocsNodeText(node)),
		})
	case isMutabilityOverlayDocsHeading(node):
		return
	default:
		if fieldName, ok := appendMutabilityOverlayParsedDocsField(node, prefix, result); ok {
			parseMutabilityOverlayDocsDescendantLists(node, appendClonedStrings(prefix, fieldName), result)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			parseMutabilityOverlayDocsSectionNode(child, prefix, result)
		}
	}
}

func parseMutabilityOverlayDocsListNode(node *xhtml.Node, prefix []string, result *mutabilityOverlayDocsParseResult) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode && strings.EqualFold(child.Data, "li") {
			parseMutabilityOverlayDocsListItem(child, prefix, result)
		}
	}
}

func parseMutabilityOverlayDocsListItem(node *xhtml.Node, prefix []string, result *mutabilityOverlayDocsParseResult) {
	fieldName, ok := appendMutabilityOverlayParsedDocsField(node, prefix, result)
	if !ok {
		appendMutabilityOverlayDocsDiagnostic(result, mutabilityOverlayDocsParserDiagnostic{
			Severity:      mutabilityOverlayDocsParseDiagnosticSeverityError,
			Reason:        mutabilityOverlayDocsParseDiagnosticUnsupportedFieldEntry,
			FieldPath:     strings.Join(prefix, "."),
			SectionAnchor: result.SectionAnchor,
			Detail:        "Argument Reference list item did not expose a leading field token",
			EvidenceText:  normalizeMutabilityOverlayDocsText(mutabilityOverlayDocsDirectTextWithoutLists(node)),
		})
		parseMutabilityOverlayDocsDescendantLists(node, prefix, result)
		return
	}

	parseMutabilityOverlayDocsDescendantLists(node, appendClonedStrings(prefix, fieldName), result)
}

func parseMutabilityOverlayDocsDescendantLists(node *xhtml.Node, prefix []string, result *mutabilityOverlayDocsParseResult) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode {
			continue
		}
		switch {
		case isMutabilityOverlayDocsList(child):
			parseMutabilityOverlayDocsListNode(child, prefix, result)
		case strings.EqualFold(child.Data, "table"):
			appendMutabilityOverlayDocsDiagnostic(result, mutabilityOverlayDocsParserDiagnostic{
				Severity:      mutabilityOverlayDocsParseDiagnosticSeverityError,
				Reason:        mutabilityOverlayDocsParseDiagnosticUnsupportedSectionNode,
				FieldPath:     strings.Join(prefix, "."),
				SectionAnchor: result.SectionAnchor,
				Detail:        "Argument Reference field entry uses an unsupported <table> layout",
				EvidenceText:  normalizeMutabilityOverlayDocsText(mutabilityOverlayDocsNodeText(child)),
			})
		default:
			parseMutabilityOverlayDocsDescendantLists(child, prefix, result)
		}
	}
}

func appendMutabilityOverlayParsedDocsField(node *xhtml.Node, prefix []string, result *mutabilityOverlayDocsParseResult) (string, bool) {
	fieldName, evidenceText := extractMutabilityOverlayDocsFieldEntry(node)
	if fieldName == "" {
		return "", false
	}

	fieldPath := mutabilityOverlayJoinDocsFieldPath(prefix, fieldName)
	state, detail, rawSignal := classifyMutabilityOverlayDocsFieldEvidence(evidenceText)
	result.Fields = append(result.Fields, mutabilityOverlayParsedDocsEvidence{
		FieldPath:     fieldPath,
		EvidenceState: state,
		Detail:        detail,
		RawSignal:     rawSignal,
		Provenance: mutabilityOverlayDocsEvidenceLocation{
			RegistryURL:   result.RegistryURL,
			SectionTitle:  result.SectionTitle,
			SectionAnchor: result.SectionAnchor,
			EvidenceText:  evidenceText,
		},
	})
	return fieldName, true
}

func extractMutabilityOverlayDocsFieldEntry(node *xhtml.Node) (string, string) {
	evidenceText := normalizeMutabilityOverlayDocsText(mutabilityOverlayDocsDirectTextWithoutLists(node))
	fieldName := normalizeMutabilityOverlayDocsText(mutabilityOverlayDocsFirstCodeTextWithoutLists(node))
	if fieldName == "" {
		fieldName = mutabilityOverlayDocsFallbackFieldToken(evidenceText)
	}
	return fieldName, evidenceText
}

func classifyMutabilityOverlayDocsFieldEvidence(evidenceText string) (string, string, string) {
	evidenceText = normalizeMutabilityOverlayDocsText(evidenceText)
	lower := strings.ToLower(evidenceText)

	updatableSignals := append([]string(nil), collectMutabilityOverlayDocsSignals(lower, []string{
		"(updatable)",
		"can be updated",
		"updated in place",
	})...)
	deniedSignals := append([]string(nil), collectMutabilityOverlayDocsSignals(lower, []string{
		"updating this value after creation is not supported",
		"updating this field after creation is not supported",
		"updates to this field are not supported",
		"cannot be updated",
		"can't be updated",
		"does not support update",
		"not updatable",
	})...)
	replacementSignals := append([]string(nil), collectMutabilityOverlayDocsSignals(lower, []string{
		"forces a new resource",
		"force a new resource",
		"requires replacement",
		"require replacement",
		"must be recreated",
		"will be recreated",
		"recreate the resource",
		"destroyed and recreated",
		"destruction and recreation",
	})...)

	rawSignals := uniqueSortedStrings(append(append([]string(nil), updatableSignals...), append(deniedSignals, replacementSignals...)...))
	rawSignal := strings.Join(rawSignals, "; ")

	explicitConditionalReplacement := strings.Contains(lower, "may be updated in place") &&
		(strings.Contains(lower, "require replacement") || strings.Contains(lower, "requires replacement") || strings.Contains(lower, "recreated"))
	switch {
	case explicitConditionalReplacement || (len(updatableSignals) != 0 && (len(deniedSignals) != 0 || len(replacementSignals) != 0)):
		if rawSignal == "" {
			rawSignal = evidenceText
		}
		return mutabilityOverlayDocsStateAmbiguous, "Argument Reference contains conflicting or conditional update guidance.", rawSignal
	case len(replacementSignals) != 0:
		return mutabilityOverlayDocsStateReplacementRequired, "Argument Reference says changing this field requires replacement.", rawSignal
	case len(deniedSignals) != 0:
		return mutabilityOverlayDocsStateDeniedUpdatable, "Argument Reference says in-place updates are not supported.", rawSignal
	case len(updatableSignals) != 0:
		return mutabilityOverlayDocsStateConfirmedUpdatable, "Argument Reference marks the field as updatable in place.", rawSignal
	default:
		return mutabilityOverlayDocsStateUnknown, "Argument Reference documents the field but does not include an explicit update cue.", ""
	}
}

func collectMutabilityOverlayDocsSignals(lower string, phrases []string) []string {
	signals := make([]string, 0, len(phrases))
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			signals = append(signals, phrase)
		}
	}
	return uniqueSortedStrings(signals)
}

func mutabilityOverlayJoinDocsFieldPath(prefix []string, fieldName string) string {
	parts := append([]string(nil), prefix...)
	fieldName = strings.TrimSpace(fieldName)
	if fieldName != "" {
		parts = append(parts, fieldName)
	}
	return strings.Join(parts, ".")
}

func mutabilityOverlayDocsFirstCodeTextWithoutLists(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == xhtml.ElementNode && isMutabilityOverlayDocsList(node) {
		return ""
	}
	if node.Type == xhtml.ElementNode && strings.EqualFold(node.Data, "code") {
		return mutabilityOverlayDocsNodeText(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if text := mutabilityOverlayDocsFirstCodeTextWithoutLists(child); text != "" {
			return text
		}
	}
	return ""
}

func mutabilityOverlayDocsDirectTextWithoutLists(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == xhtml.ElementNode && isMutabilityOverlayDocsList(node) {
		return ""
	}
	if node.Type == xhtml.TextNode {
		return node.Data
	}

	var b strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(mutabilityOverlayDocsDirectTextWithoutLists(child))
		b.WriteByte(' ')
	}
	return b.String()
}

func mutabilityOverlayDocsFallbackFieldToken(evidenceText string) string {
	token := strings.TrimSpace(evidenceText)
	if before, _, ok := strings.Cut(token, " - "); ok {
		token = strings.TrimSpace(before)
	}
	if token == "" {
		return ""
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '.' || r == '*' || r == '[' || r == ']':
		default:
			return ""
		}
	}
	return token
}

func normalizeMutabilityOverlayDocsText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func mutabilityOverlayDocsNodeText(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == xhtml.TextNode {
		return node.Data
	}
	var b strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(mutabilityOverlayDocsNodeText(child))
		b.WriteByte(' ')
	}
	return b.String()
}

func mutabilityOverlayDocsNodeAttr(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func isMutabilityOverlayDocsList(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode {
		return false
	}
	return strings.EqualFold(node.Data, "ul") || strings.EqualFold(node.Data, "ol")
}

func isMutabilityOverlayDocsHeading(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode || len(node.Data) != 2 {
		return false
	}
	return (node.Data[0] == 'h' || node.Data[0] == 'H') && node.Data[1] >= '1' && node.Data[1] <= '6'
}

func mutabilityOverlayDocsHeadingLevel(tag string) int {
	if len(tag) != 2 || (tag[0] != 'h' && tag[0] != 'H') || tag[1] < '1' || tag[1] > '6' {
		return 0
	}
	return int(tag[1] - '0')
}

func appendMutabilityOverlayDocsDiagnostic(result *mutabilityOverlayDocsParseResult, diagnostic mutabilityOverlayDocsParserDiagnostic) {
	if result == nil {
		return
	}
	diagnostic.Detail = strings.TrimSpace(diagnostic.Detail)
	diagnostic.EvidenceText = strings.TrimSpace(diagnostic.EvidenceText)
	diagnostic.FieldPath = strings.TrimSpace(diagnostic.FieldPath)
	diagnostic.SectionAnchor = strings.TrimSpace(diagnostic.SectionAnchor)
	result.Diagnostics = append(result.Diagnostics, diagnostic)
}

func countMutabilityOverlayDocsErrorDiagnostics(diagnostics []mutabilityOverlayDocsParserDiagnostic) int {
	count := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == mutabilityOverlayDocsParseDiagnosticSeverityError {
			count++
		}
	}
	return count
}

func annotateMutabilityOverlayDocsDuplicateFieldPaths(result *mutabilityOverlayDocsParseResult) {
	if result == nil || len(result.Fields) == 0 {
		return
	}

	byPath := make(map[string][]mutabilityOverlayParsedDocsEvidence, len(result.Fields))
	for _, field := range result.Fields {
		byPath[field.FieldPath] = append(byPath[field.FieldPath], field)
	}

	paths := make([]string, 0, len(byPath))
	for path, fields := range byPath {
		if len(fields) > 1 {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	for _, fieldPath := range paths {
		fields := byPath[fieldPath]
		states := make([]string, 0, len(fields))
		for _, field := range fields {
			states = append(states, field.EvidenceState)
		}
		states = uniqueSortedStrings(states)
		appendMutabilityOverlayDocsDiagnostic(result, mutabilityOverlayDocsParserDiagnostic{
			Severity:      mutabilityOverlayDocsParseDiagnosticSeverityWarning,
			Reason:        mutabilityOverlayDocsParseDiagnosticDuplicateFieldPath,
			FieldPath:     fieldPath,
			SectionAnchor: result.SectionAnchor,
			Detail:        fmt.Sprintf("Argument Reference emitted %d rows for the same field path with states %s", len(fields), strings.Join(states, ", ")),
		})
	}
}
