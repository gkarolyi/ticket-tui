package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
)

var relatedTicketLinePattern = regexp.MustCompile(`^- ([^ ]+) \[([^]]+)\] (.+)$`)

type metadataField struct {
	label string
	value string
}

func RenderTicketDetail(raw string, ticketsDir string, widths ...int) string {
	width := 0
	if len(widths) > 0 {
		width = widths[0]
	}
	frontmatter, body := splitFrontmatter(raw)
	fields := metadataFields(frontmatter)
	sections := make([]string, 0, 3)
	if metadata := renderMetadata(fields, width); metadata != "" {
		sections = append(sections, metadata)
	}
	if graph := renderRelationshipGraph(frontmatter, body); graph != "" {
		sections = append(sections, graph)
	}
	if preview := renderMarkdownPreview(body, ticketsDir, width); preview != "" {
		sections = append(sections, preview)
	}
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func splitFrontmatter(raw string) ([]string, string) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, raw
	}

	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return lines[1:i], strings.Join(lines[i+1:], "\n")
		}
	}
	return nil, raw
}

func metadataFields(lines []string) []metadataField {
	fields := make([]metadataField, 0, len(lines))
	for _, line := range lines {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields = append(fields, metadataField{
			label: titleCase(strings.TrimSpace(key)),
			value: formatMetadataValue(strings.TrimSpace(value)),
		})
	}
	return fields
}

func renderMetadata(fields []metadataField, width int) string {
	if len(fields) == 0 {
		return ""
	}

	lines := []string{titleStyle.Render("Metadata")}
	for _, group := range metadataGroups(fields, width) {
		headers := make([]string, 0, len(group))
		values := make([]string, 0, len(group))
		for _, field := range group {
			colWidth := metadataColumnWidth(field)
			headers = append(headers, padRight(field.label, colWidth))
			values = append(values, padRight(truncateCell(field.value, colWidth), colWidth))
		}
		lines = append(lines, strings.TrimRight(strings.Join(headers, "  "), " "))
		lines = append(lines, strings.TrimRight(strings.Join(values, "  "), " "))
	}
	return strings.Join(lines, "\n")
}

func metadataGroups(fields []metadataField, width int) [][]metadataField {
	if width <= 0 {
		return [][]metadataField{fields}
	}

	groups := make([][]metadataField, 0, len(fields))
	current := make([]metadataField, 0, len(fields))
	currentWidth := 0
	for _, field := range fields {
		colWidth := metadataColumnWidth(field)
		required := colWidth
		if len(current) > 0 {
			required += 2
		}
		if len(current) > 0 && currentWidth+required > width {
			groups = append(groups, current)
			current = []metadataField{field}
			currentWidth = colWidth
			continue
		}
		current = append(current, field)
		currentWidth += required
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

func metadataColumnWidth(field metadataField) int {
	width := max(len(field.label), len(field.value))
	if width < 8 {
		return 8
	}
	if width > 18 {
		return 18
	}
	return width
}

func truncateCell(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width == 1 {
		return value[:1]
	}
	return value[:width-1] + "…"
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func formatMetadataValue(value string) string {
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return "-"
	}
	return value
}

func titleCase(key string) string {
	if key == "" {
		return ""
	}
	return strings.ToUpper(key[:1]) + key[1:]
}

func renderMarkdownPreview(raw string, ticketsDir string, width int) string {
	if width > 0 {
		if rendered, err := renderGlamourMarkdown(raw, width); err == nil {
			return strings.TrimSpace(rendered)
		}
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## "):
			lines[i] = titleStyle.Render(strings.TrimPrefix(trimmed, "## "))
		case strings.HasPrefix(trimmed, "# "):
			lines[i] = titleStyle.Render(strings.TrimPrefix(trimmed, "# "))
		case relatedTicketLinePattern.MatchString(trimmed):
			lines[i] = renderRelatedTicketLine(trimmed, ticketsDir)
		case width > 0:
			lines[i] = wrapLine(line, width)
		}
	}
	return strings.Join(lines, "\n")
}

func renderGlamourMarkdown(raw string, width int) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourStyleName()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(raw)
}

func glamourStyleName() string {
	return "dark"
}

func wrapLine(line string, width int) string {
	if width <= 0 || len(line) <= width {
		return line
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}
	wrapped := make([]string, 0, len(words))
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			wrapped = append(wrapped, current)
			current = word
			continue
		}
		current += " " + word
	}
	wrapped = append(wrapped, current)
	return strings.Join(wrapped, "\n")
}

func renderRelationshipGraph(frontmatter []string, body string) string {
	lines := []string{titleStyle.Render("Relationships")}
	for _, line := range frontmatter {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(key) == "parent" {
			parent := strings.TrimSpace(value)
			if parent != "" {
				lines = append(lines, "parent <- "+parent)
			}
		}
	}

	section := ""
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			section = strings.TrimPrefix(trimmed, "## ")
			continue
		}
		if !relatedTicketLinePattern.MatchString(trimmed) {
			continue
		}
		matches := relatedTicketLinePattern.FindStringSubmatch(trimmed)
		if len(matches) != 4 {
			continue
		}
		id := matches[1]
		status := "[" + matches[2] + "]"
		detail := strings.TrimSpace(id + " " + status + " " + matches[3])
		switch section {
		case "Blockers":
			lines = append(lines, "blocked by <- "+detail)
		case "Blocking":
			lines = append(lines, "blocks -> "+detail)
		case "Children":
			lines = append(lines, "child -> "+detail)
		case "Linked":
			lines = append(lines, "linked -- "+detail)
		}
	}

	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func renderRelatedTicketLine(line string, ticketsDir string) string {
	matches := relatedTicketLinePattern.FindStringSubmatch(line)
	if len(matches) != 4 || ticketsDir == "" {
		return line
	}
	id := matches[1]
	title := matches[3]
	path := filepath.Join(ticketsDir, id+".md")
	link := fmt.Sprintf("\x1b]8;;file://%s\x1b\\%s\x1b]8;;\x1b\\", path, title)
	return strings.Replace(line, title, link, 1)
}
