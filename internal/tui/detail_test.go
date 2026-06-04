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
	if !strings.Contains(rendered, "Metadata") {
		t.Fatalf("rendered detail missing Metadata section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Status: open") {
		t.Fatalf("rendered detail missing formatted status:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Tags: ui, backend") {
		t.Fatalf("rendered detail missing formatted tags:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Build UI") {
		t.Fatalf("rendered detail missing markdown heading:\n%s", rendered)
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
		"Relationships",
		"parent <- parent-0001",
		"blocked by <- dep-0001 [open] Dependency title",
		"blocks -> blocked-0001 [open] Blocked title",
		"child -> grand-0001 [open] Grandchild title",
		"linked -- link-0001 [open] Related title",
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
