package tui

import (
	"regexp"
	"strings"
	"testing"
)

func TestRenderTicketDetailFormatsFrontmatter(t *testing.T) {
	raw := `---
id: abc-1234
status: open
deps: []
links: []
priority: 2
tags: [ui, backend]
---
# Build UI

## Acceptance Criteria

- Works well
`

	rendered := RenderTicketDetail(raw, "/tmp/.tickets")

	if strings.Contains(rendered, "---") {
		t.Fatalf("rendered detail still contains raw frontmatter separators:\n%s", rendered)
	}
	for _, want := range []string{"Status", "Priority", "open", "Build UI"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered detail missing metadata value %q:\n%s", want, rendered)
		}
	}
}

func TestRenderTicketDetailFormatsMetadataAsHorizontalGrid(t *testing.T) {
	raw := `---
status: open
priority: 2
assignee: gkarolyi
tags: [ui, backend]
created: 2026-05-28
---
# Build UI
`

	rendered := stripDetailANSI(RenderTicketDetail(raw, "/tmp/.tickets", 80))

	for _, want := range []string{"Status", "Priority", "Assignee", "Created"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("missing metadata header %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "Status: open") {
		t.Fatalf("detail still uses row-style metadata:\n%s", rendered)
	}
	if !strings.Contains(rendered, "open") || !strings.Contains(rendered, "gkarolyi") {
		t.Fatalf("detail missing metadata values:\n%s", rendered)
	}
}

func TestRenderTicketDetailWrapsMetadataIntoCompactGroupsAtNarrowWidth(t *testing.T) {
	raw := `---
status: in_progress
priority: 1
assignee: gkarolyi
tags: [tui, ux]
created: 2026-05-28
---
# Build UI
`

	rendered := stripDetailANSI(RenderTicketDetail(raw, "/tmp/.tickets", 36))

	if !strings.Contains(rendered, "Status") || !strings.Contains(rendered, "Priority") {
		t.Fatalf("narrow detail missing metadata groups:\n%s", rendered)
	}
	if strings.Contains(rendered, "Status: in_progress") {
		t.Fatalf("narrow detail fell back to old row layout:\n%s", rendered)
	}
}

func TestRenderTicketDetailShowsTitleBeforeMetadata(t *testing.T) {
	raw := `---
id: abc-1234
status: open
deps: []
links: []
priority: 2
assignee: gkarolyi
tags: [ui, backend]
created: 2026-05-28
type: feature
---
# Build UI

Summary paragraph.
`

	rendered := stripDetailANSI(RenderTicketDetail(raw, "/tmp/.tickets", 80))

	if strings.Index(rendered, "Build UI") > strings.Index(rendered, "Status") {
		t.Fatalf("ticket title appears after metadata:\n%s", rendered)
	}
}

func TestRenderTicketDetailKeepsMetadataGridCompact(t *testing.T) {
	raw := `---
id: abc-1234
status: open
deps: []
links: []
priority: 2
assignee: gkarolyi
tags: [ui, backend]
created: 2026-05-28
type: feature
---
# Build UI
`

	rendered := stripDetailANSI(RenderTicketDetail(raw, "/tmp/.tickets", 80))

	if strings.Contains(rendered, "Id") || strings.Contains(rendered, "Deps") || strings.Contains(rendered, "Links") || strings.Contains(rendered, "Type") {
		t.Fatalf("metadata grid includes low-value fields that should not be in the compact summary:\n%s", rendered)
	}
}

func TestRenderTicketDetailLinksRelatedTickets(t *testing.T) {
	raw := `# Blocked ticket

## Blockers

- abc-0001 [open] Dependency title
`

	rendered := RenderTicketDetail(raw, "/tmp/.tickets")

	if !strings.Contains(rendered, "\x1b]8;;file:///tmp/.tickets/abc-0001.md\x1b\\") {
		t.Fatalf("rendered detail missing terminal hyperlink:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Dependency title") {
		t.Fatalf("rendered detail missing dependency title:\n%s", rendered)
	}
}

func TestRenderTicketDetailPartsSplitPinnedHeaderAndBody(t *testing.T) {
	raw := `---
id: abc-1234
status: open
deps: []
links: []
priority: 2
assignee: gkarolyi
tags: [ui, backend]
created: 2026-05-28T14:45:56Z
type: feature
---
# Build UI

Summary paragraph.

## Acceptance Criteria

- Works well
`

	rendered := RenderTicketDetailParts(raw, "/tmp/.tickets", 60)

	for _, want := range []string{"Build UI", "Status", "Priority", "Created", "2026-05-28"} {
		if !strings.Contains(rendered.Header, want) {
			t.Fatalf("header missing %q:\n%s", want, rendered.Header)
		}
	}
	body := stripDetailANSI(rendered.Body)
	if strings.Contains(body, "Status") || strings.Contains(body, "Created") {
		t.Fatalf("body still contains pinned metadata:\n%s", body)
	}
	if !strings.Contains(body, "Summary paragraph") {
		t.Fatalf("body missing markdown content:\n%s", body)
	}
}

func TestRenderTicketDetailShowsRelationshipGraph(t *testing.T) {
	raw := `---
id: child-0001
status: open
parent: parent-0001
deps: [dep-0001]
links: []
---
# Child ticket

## Blockers

- dep-0001 [open] Dependency title

## Blocking

- blocked-0001 [open] Blocked title

## Children

- grand-0001 [open] Grandchild title

## Linked

- link-0001 [open] Related title
`

	rendered := RenderTicketDetail(raw, "/tmp/.tickets")

	for _, want := range []string{
		"Parent      parent-0001",
		"Blocked by  dep-0001 [open] Dependency title",
		"Blocking    blocked-0001 [open] Blocked title",
		"Children    grand-0001 [open] Grandchild title",
		"Linked      link-0001 [open] Related title",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered detail missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderTicketDetailWrapsLongLinesToWidth(t *testing.T) {
	raw := `---
id: abc-1234
---
# Build UI

alpha beta gamma delta epsilon zeta
`

	rendered := stripDetailANSI(RenderTicketDetail(raw, "/tmp/.tickets", 12))

	if strings.Contains(rendered, "alpha beta gamma delta epsilon zeta") {
		t.Fatalf("rendered detail contains unwrapped long line:\n%s", rendered)
	}
	for _, want := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered detail missing wrapped word %q:\n%s", want, rendered)
		}
	}
}

func TestGlamourPreviewUsesFixedStyleToAvoidTerminalProbe(t *testing.T) {
	if glamourStyleName() == "auto" {
		t.Fatal("glamour preview uses auto style, which can emit terminal probes that are read back as TUI input")
	}
}

func stripDetailANSI(value string) string {
	value = regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(value, "")
	return regexp.MustCompile(`\x1b]8;;[^\x1b]*\x1b\\|\x1b]8;;\x1b\\`).ReplaceAllString(value, "")
}
