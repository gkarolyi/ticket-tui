package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	teatest "github.com/charmbracelet/x/exp/teatest/v2"
)

func TestTeatestEnterSwitchesToPreviewFocus(t *testing.T) {
	tm := newTeatestApp(t, 80, 24)
	t.Cleanup(func() { _ = tm.Quit() })

	time.Sleep(300 * time.Millisecond)
	tm.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(150 * time.Millisecond)
	_ = tm.Quit()

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if final.focus != focusPreview {
		t.Fatalf("focus = %v, want focusPreview", final.focus)
	}
	if final.width != 80 || final.height != 24 {
		t.Fatalf("size = %dx%d, want 80x24", final.width, final.height)
	}
}

func TestTeatestQuestionMarkOpensHelpOverlay(t *testing.T) {
	tm := newTeatestApp(t, 100, 28)
	t.Cleanup(func() { _ = tm.Quit() })

	time.Sleep(300 * time.Millisecond)
	tm.Send(keyMsg("?"))
	time.Sleep(150 * time.Millisecond)
	_ = tm.Quit()

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if !final.helpShown {
		t.Fatal("helpShown is false, want true")
	}
	if final.width != 100 || final.height != 28 {
		t.Fatalf("size = %dx%d, want 100x28", final.width, final.height)
	}
}

func TestTeatestCtrlPOpensCommandPalette(t *testing.T) {
	tm := newTeatestApp(t, 100, 28)
	t.Cleanup(func() { _ = tm.Quit() })

	time.Sleep(300 * time.Millisecond)
	tm.Send(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	time.Sleep(150 * time.Millisecond)
	_ = tm.Quit()

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if !final.palette.shown {
		t.Fatal("palette.shown is false, want true")
	}
	if final.width != 100 || final.height != 28 {
		t.Fatalf("size = %dx%d, want 100x28", final.width, final.height)
	}
}

func TestTeatestFilterNarrowsTicketsByDescription(t *testing.T) {
	tm := newTeatestApp(t, 100, 28)
	t.Cleanup(func() { _ = tm.Quit() })

	time.Sleep(300 * time.Millisecond)
	tm.Send(keyMsg("/"))
	for _, key := range []string{"s", "c", "a", "n", "n"} {
		tm.Send(keyMsg(key))
	}
	tm.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(300 * time.Millisecond)
	_ = tm.Quit()

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if final.searchFilter != "scann" {
		t.Fatalf("searchFilter = %q, want scann", final.searchFilter)
	}
	assertIDs(t, final.tickets, []string{"tic-work"})
	view := stripANSI(final.View().Content)
	if !strings.Contains(view, "metadata grid") {
		t.Fatalf("filtered view missing matching ticket:\n%s", view)
	}
	if strings.Contains(view, "dashboard header") || strings.Contains(view, "blocked by dependency") {
		t.Fatalf("filtered view still includes non-matching tickets:\n%s", view)
	}
}

func TestTeatestEmptyFilterSubmitClearsFilter(t *testing.T) {
	tm := newTeatestApp(t, 100, 28)
	t.Cleanup(func() { _ = tm.Quit() })

	time.Sleep(300 * time.Millisecond)
	tm.Send(keyMsg("/"))
	for _, key := range []string{"s", "c", "a", "n", "n"} {
		tm.Send(keyMsg(key))
	}
	tm.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(150 * time.Millisecond)
	tm.Send(keyMsg("/"))
	tm.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(300 * time.Millisecond)
	_ = tm.Quit()

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if final.searchFilter != "" {
		t.Fatalf("searchFilter = %q, want empty", final.searchFilter)
	}
	assertIDs(t, final.tickets, []string{"tic-ready", "tic-work", "tic-block", "tic-done"})
}

func newTeatestApp(t *testing.T, width int, height int) *teatest.TestModel {
	t.Helper()
	dir := t.TempDir()
	writeFixtureTicket(t, dir, "tic-ready.md", `---
id: tic-ready
status: open
deps: []
links: []
priority: 1
assignee: gkarolyi
tags: [ui, ux]
created: 2026-05-28
type: feature
---
# dashboard header

Design a clearer dashboard-first UX for ticket-tui.

## Acceptance

- header counts visible
- queue remains readable
`)
	writeFixtureTicket(t, dir, "tic-work.md", `---
id: tic-work
status: in_progress
deps: []
links: []
priority: 1
assignee: gkarolyi
tags: [tui, ux]
created: 2026-05-27
type: task
---
# metadata grid

Keep metadata compact and scannable.
`)
	writeFixtureTicket(t, dir, "tic-block.md", `---
id: tic-block
status: open
deps: [tic-work]
links: []
priority: 2
assignee: gkarolyi
tags: [dependency]
created: 2026-05-26
type: task
---
# blocked by dependency

## Blockers

- tic-work [in_progress] metadata grid
`)
	writeFixtureTicket(t, dir, "tic-done.md", `---
id: tic-done
status: closed
deps: []
links: []
priority: 3
assignee: gkarolyi
tags: [done]
created: 2026-05-25
type: chore
---
# closed example

Already complete.
`)

	runner := func(name string, args ...string) (string, error) {
		id := args[len(args)-1]
		content, err := os.ReadFile(filepath.Join(dir, id+".md"))
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	m := newModel(Config{TicketsDir: dir, TKScript: "/usr/local/bin/tk"}, runner)
	m.width = width
	m.height = height
	m.resizeDetail()
	return teatest.NewTestModel(t, m, teatest.WithInitialTermSize(width, height))
}

func writeFixtureTicket(t *testing.T, dir string, name string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
