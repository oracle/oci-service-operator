package plantuml

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const batchSize = 200

// Artifact identifies one checked-in PlantUML source file and its rendered SVG.
type Artifact struct {
	SourcePath   string
	SourceData   []byte
	RenderedPath string
}

// Pair identifies one checked-in PlantUML source/SVG pair that should stay in sync.
type Pair struct {
	SourcePath   string
	RenderedPath string
}

// Binary returns the plantuml executable path.
func Binary() (string, error) {
	bin, err := exec.LookPath("plantuml")
	if err != nil {
		return "", fmt.Errorf("plantuml binary not found in PATH")
	}
	return bin, nil
}

// RenderArtifacts renders the provided PlantUML sources to SVG under root.
func RenderArtifacts(root string, artifacts []Artifact) error {
	if len(artifacts) == 0 {
		return nil
	}

	bin, err := Binary()
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, filepath.Join(root, filepath.FromSlash(artifact.SourcePath)))
	}

	for start := 0; start < len(paths); start += batchSize {
		end := start + batchSize
		if end > len(paths) {
			end = len(paths)
		}

		args := append([]string{"--svg", "--overwrite", "--skip-fresh"}, paths[start:end]...)
		cmd := exec.Command(bin, args...)
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(output))
			if detail == "" {
				return fmt.Errorf("render PlantUML SVGs: %w", err)
			}
			return fmt.Errorf("render PlantUML SVGs: %w: %s", err, detail)
		}
	}

	return nil
}

// ValidateRenderedArtifacts checks that each SVG embeds source metadata matching its .puml file.
func ValidateRenderedArtifacts(root string, pairs []Pair) []string {
	if len(pairs) == 0 {
		return nil
	}

	bin, err := Binary()
	if err != nil {
		return []string{err.Error()}
	}

	renderedPaths := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		renderedPaths = append(renderedPaths, filepath.Join(root, filepath.FromSlash(pair.RenderedPath)))
	}

	extracted := make(map[string]string, len(pairs))
	for start := 0; start < len(renderedPaths); start += batchSize {
		end := start + batchSize
		if end > len(renderedPaths) {
			end = len(renderedPaths)
		}

		decoded, err := extractSources(bin, renderedPaths[start:end])
		if err != nil {
			return []string{err.Error()}
		}
		for fullPath, source := range decoded {
			extracted[filepath.Clean(fullPath)] = source
		}
	}

	var problems []string
	for _, pair := range pairs {
		sourcePath := filepath.Join(root, filepath.FromSlash(pair.SourcePath))
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", filepath.ToSlash(pair.SourcePath), err))
			continue
		}

		renderedPath := filepath.Join(root, filepath.FromSlash(pair.RenderedPath))
		embedded, ok := extracted[filepath.Clean(renderedPath)]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: expected embedded PlantUML source metadata", filepath.ToSlash(pair.RenderedPath)))
			continue
		}

		if normalizeSource(source) != embedded {
			problems = append(problems, fmt.Sprintf("%s: stale rendered artifact; run `make formal-diagrams`", filepath.ToSlash(pair.RenderedPath)))
		}
	}

	return problems
}

// BaseHeader returns a common styled PlantUML header.
func BaseHeader(title string) []string {
	return []string{
		"@startuml",
		"skinparam shadowing false",
		"skinparam backgroundColor #fcfcfd",
		"skinparam defaultFontName Helvetica",
		"skinparam defaultFontSize 14",
		"skinparam arrowColor #0f172a",
		"skinparam noteBackgroundColor #fff7d6",
		"skinparam noteBorderColor #b45309",
		"skinparam noteFontColor #334155",
		"skinparam roundcorner 16",
		fmt.Sprintf("title %s", title),
	}
}

// ActivityHeader returns a styled PlantUML activity header.
func ActivityHeader(title string) []string {
	return append(BaseHeader(title),
		"skinparam activity {",
		"  backgroundColor #f8fafc",
		"  borderColor #0f172a",
		"  fontColor #0f172a",
		"  diamondBackgroundColor #e2e8f0",
		"  diamondBorderColor #1d4ed8",
		"  diamondFontColor #0f172a",
		"  startColor #1d4ed8",
		"  endColor #0f172a",
		"  barColor #1d4ed8",
		"}",
	)
}

// SequenceHeader returns a styled PlantUML sequence header.
func SequenceHeader(title string) []string {
	return append(BaseHeader(title),
		"hide footbox",
		"skinparam sequence {",
		"  actorBorderColor #0f172a",
		"  actorBackgroundColor #e2e8f0",
		"  participantBorderColor #0f172a",
		"  participantBackgroundColor #e2e8f0",
		"  participantFontColor #0f172a",
		"  lifeLineBorderColor #94a3b8",
		"  groupBorderColor #cbd5e1",
		"  groupBackgroundColor #f8fafc",
		"}",
	)
}

// StateHeader returns a styled PlantUML state-machine header.
func StateHeader(title string) []string {
	return append(BaseHeader(title),
		"skinparam state {",
		"  backgroundColor #f8fafc",
		"  borderColor #0f172a",
		"  fontColor #0f172a",
		"  startColor #1d4ed8",
		"  endColor #0f172a",
		"}",
	)
}

// Action formats a wrapped PlantUML activity action.
func Action(text string) string {
	return fmt.Sprintf(":%s;", WrapText(text, 46))
}

// WrapText wraps plain text and joins lines using PlantUML line breaks.
func WrapText(text string, limit int) string {
	lines := WrapNoteLines(limit, text)
	if len(lines) == 0 {
		return "none"
	}
	return strings.Join(lines, `\n`)
}

// WrapNoteLines wraps one or more values for PlantUML note bodies.
func WrapNoteLines(limit int, values ...string) []string {
	var lines []string
	for _, value := range values {
		for _, segment := range strings.Split(strings.TrimSpace(value), "\n") {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				continue
			}
			lines = append(lines, wrapLine(segment, limit)...)
		}
	}
	if len(lines) == 0 {
		return []string{"none"}
	}
	return lines
}

func extractSources(bin string, renderedPaths []string) (map[string]string, error) {
	if len(renderedPaths) == 0 {
		return map[string]string{}, nil
	}

	args := append([]string{"--extract-source"}, renderedPaths...)
	cmd := exec.Command(bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			return nil, fmt.Errorf("extract PlantUML sources: %w", err)
		}
		return nil, fmt.Errorf("extract PlantUML sources: %w: %s", err, detail)
	}

	return parseExtractOutput(string(output))
}

func parseExtractOutput(output string) (map[string]string, error) {
	decoded := make(map[string]string)
	for _, block := range strings.Split(output, "------------------------") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		fullPath := filepath.Clean(strings.TrimSpace(lines[0]))
		if fullPath == "" {
			return nil, fmt.Errorf("extract PlantUML sources: missing SVG path in output block")
		}

		index := 1
		for index < len(lines) && strings.TrimSpace(lines[index]) == "" {
			index++
		}

		source := strings.Join(lines[index:], "\n")
		if source != "" && !strings.HasSuffix(source, "\n") {
			source += "\n"
		}
		decoded[fullPath] = normalizeSource([]byte(source))
	}

	return decoded, nil
}

func normalizeSource(source []byte) string {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return text + "\n"
}

func wrapLine(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if limit <= 0 || len(text) <= limit {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	lines := []string{words[0]}
	for _, word := range words[1:] {
		current := lines[len(lines)-1]
		if len(current)+1+len(word) <= limit {
			lines[len(lines)-1] = current + " " + word
			continue
		}
		lines = append(lines, word)
	}

	return lines
}
