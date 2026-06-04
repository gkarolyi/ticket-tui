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

type detailParts struct {
	Header string
	Body   string
}

func RenderTicketDetail(raw string, ticketsDir string, widths ...int) string {
	parts := RenderTicketDetailParts(raw, ticketsDir, widths...)
	sections := make([]string, 0, 2)
	if parts.Header != "" {
		sections = append(sections, parts.Header)
	}
	if parts.Body != "" {
		sections = append(sections, parts.Body)
	}
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func RenderTicketDetailParts(raw string, ticketsDir string, widths ...int) detailParts {
	width := 0
	if len(widths) > 0 {
		width = widths[0]
	}
	frontmatter, body := splitFrontmatter(raw)
	title, body := extractTitle(body)
	fields := metadataFields(frontmatter)
	headerSections := make([]string, 0, 3)
	if title != "" {
		headerSections = append(headerSections, titleStyle.Render(title))
	}
	if metadata := renderMetadata(fields, width); metadata != "" {
		headerSections = append(headerSections, metadata)
	}
	if graph := renderRelationshipGraph(frontmatter, body); graph != "" {
		headerSections = append(headerSections, graph)
	}
	return detailParts{
		Header: strings.TrimSpace(strings.Join(headerSections, "\n\n")),
		Body:   renderMarkdownPreview(body, ticketsDir, width),
	}
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

func extractTitle(body string) (string, string) {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			remaining := append([]string{}, lines[:i]...)
			remaining = append(remaining, lines[i+1:]...)
			return title, strings.TrimSpace(strings.Join(remaining, "\n"))
		}
	}
	return "", body
}

func metadataFields(lines []string) []metadataField {
	values := make(map[string]string, len(lines))
	for _, line := range lines {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		values[key] = formatMetadataValue(key, strings.TrimSpace(value))
	}
	ordered := []struct {
		key   string
		label string
	}{
		{key: "status", label: "Status"},
		{key: "priority", label: "Priority"},
		{key: "assignee", label: "Assignee"},
		{key: "created", label: "Created"},
	}
	fields := make([]metadataField, 0, len(ordered))
	for _, item := range ordered {
		value, ok := values[item.key]
		if !ok {
			continue
		}
		fields = append(fields, metadataField{label: item.label, value: value})
	}
	return fields
}

func renderMetadata(fields []metadataField, width int) string {
	if len(fields) == 0 {
		return ""
	}

	lines := make([]string, 0, len(fields)*2)
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

func formatMetadataValue(key string, value string) string {
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if key == "created" && len(value) >= len("2006-01-02") {
		return value[:10]
	}
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
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return ""
	}
	if width > 0 {
		if rendered, err := renderGlamourMarkdown(trimmedRaw, width); err == nil {
			return strings.TrimSpace(rendered)
		}
	}

	lines := strings.Split(trimmedRaw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## "):
			lines[i] = titleStyle.Render(strings.TrimPrefix(trimmed, "## "))
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
	lines := make([]string, 0, 8)
	for _, line := range frontmatter {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(key) == "parent" {
			parent := strings.TrimSpace(value)
			if parent != "" {
				lines = append(lines, "Parent      "+parent)
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
			lines = append(lines, "Blocked by  "+detail)
		case "Blocking":
			lines = append(lines, "Blocking    "+detail)
		case "Children":
			lines = append(lines, "Children    "+detail)
		case "Linked":
			lines = append(lines, "Linked      "+detail)
		}
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
