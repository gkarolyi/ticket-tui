package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewConfigRequiresTicketsDir(t *testing.T) {
	_, err := NewConfig(map[string]string{"PWD": t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing TICKETS_DIR")
	}
}

func TestNewConfigFindsTicketsDirFromCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	ticketsDir := filepath.Join(dir, ".tickets")
	t.Setenv("PWD", dir)

	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	config, err := NewConfig(map[string]string{"PWD": dir})
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}
	if config.TicketsDir != ticketsDir {
		t.Fatalf("TicketsDir = %q, want %q", config.TicketsDir, ticketsDir)
	}
}

func TestNewConfigAcceptsTicketsDirAndScript(t *testing.T) {
	config, err := NewConfig(map[string]string{
		"TICKETS_DIR": "/tmp/tickets",
		"TK_SCRIPT":   "/usr/local/bin/tk",
	})
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}
	if config.TicketsDir != "/tmp/tickets" {
		t.Fatalf("TicketsDir = %q", config.TicketsDir)
	}
	if config.TKScript != "/usr/local/bin/tk" {
		t.Fatalf("TKScript = %q", config.TKScript)
	}
}

func TestSelectedIDReturnsEmptyWithoutSelection(t *testing.T) {
	id := selectedID(model{})
	if id != "" {
		t.Fatalf("selectedID = %q", id)
	}
}

func TestEditProcessUsesTkScriptAndTerminalStdio(t *testing.T) {
	cmd, err := editProcess(Config{TKScript: "/usr/local/bin/tk"}, "abc-1234")
	if err != nil {
		t.Fatalf("editProcess returned error: %v", err)
	}

	if cmd.Path != "/usr/local/bin/tk" {
		t.Fatalf("Path = %q", cmd.Path)
	}
	assertStrings(t, cmd.Args, []string{"/usr/local/bin/tk", "edit", "abc-1234"})
	if cmd.Stdin == nil {
		t.Fatal("Stdin is nil")
	}
	if cmd.Stdout == nil {
		t.Fatal("Stdout is nil")
	}
	if cmd.Stderr == nil {
		t.Fatal("Stderr is nil")
	}
}

func TestCreateCommandUsesSuperCreate(t *testing.T) {
	cmd, err := commandSpec(Config{TKScript: "/usr/local/bin/tk"}, "super", "create", "New ticket")
	if err != nil {
		t.Fatalf("commandSpec returned error: %v", err)
	}
	if cmd.name != "/usr/local/bin/tk" {
		t.Fatalf("name = %q", cmd.name)
	}
	assertStrings(t, cmd.args, []string{"super", "create", "New ticket"})
}

func TestQueryCommandUsesQueryPlugin(t *testing.T) {
	cmd, err := commandSpec(Config{TKScript: "/usr/local/bin/tk"}, "query", `.status == "open"`)
	if err != nil {
		t.Fatalf("commandSpec returned error: %v", err)
	}
	if cmd.name != "/usr/local/bin/tk" {
		t.Fatalf("name = %q", cmd.name)
	}
	assertStrings(t, cmd.args, []string{"query", `.status == "open"`})
}

func TestCreateKeyShowsModal(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 80
	m.height = 24

	updated, _ := m.Update(keyMsg("n"))
	view := updated.(model).View().Content

	if !strings.Contains(view, "Create Ticket") {
		t.Fatalf("view does not contain create modal title:\n%s", view)
	}
	if !strings.Contains(view, "Title") {
		t.Fatalf("view does not contain title label:\n%s", view)
	}
}

func TestLoadedStartupDoesNotOpenQueryPrompt(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30

	updated, _ := m.Update(loadedMsg{
		tickets: []Ticket{{ID: "tic-one", Title: "One", Status: "open", Priority: 1}},
		detail:  detailParts{Body: "# One"},
	})
	view := updated.(model).View().Content

	if updated.(model).prompt != promptNone {
		t.Fatalf("prompt = %v, want promptNone", updated.(model).prompt)
	}
	if strings.Contains(view, "Query Tickets") {
		t.Fatalf("startup view opened query prompt:\n%s", view)
	}
}

func TestLayoutUsesSideBySidePanesWhenWide(t *testing.T) {
	layout := layoutFor(120, 30)

	if layout.vertical {
		t.Fatal("layout is vertical, want side-by-side")
	}
	if layout.listWidth != 40 {
		t.Fatalf("listWidth = %d, want 40", layout.listWidth)
	}
	if layout.detailWidth != 78 {
		t.Fatalf("detailWidth = %d, want 78", layout.detailWidth)
	}
	if layout.gutterWidth != 2 {
		t.Fatalf("gutterWidth = %d, want 2", layout.gutterWidth)
	}
	if layout.bodyHeight != 27 {
		t.Fatalf("bodyHeight = %d, want 27", layout.bodyHeight)
	}
}

func TestLayoutStacksPanesWhenNarrow(t *testing.T) {
	layout := layoutFor(70, 24)

	if !layout.vertical {
		t.Fatal("layout is side-by-side, want vertical")
	}
	if layout.listWidth != 70 {
		t.Fatalf("listWidth = %d, want 70", layout.listWidth)
	}
	if layout.detailWidth != 70 {
		t.Fatalf("detailWidth = %d, want 70", layout.detailWidth)
	}
	if layout.listHeight != 8 {
		t.Fatalf("listHeight = %d, want 8", layout.listHeight)
	}
	if layout.detailHeight != 13 {
		t.Fatalf("detailHeight = %d, want 13", layout.detailHeight)
	}
}

func TestLayoutUsesSideBySidePanesAtHundredColumns(t *testing.T) {
	layout := layoutFor(100, 28)

	if layout.vertical {
		t.Fatal("layout is vertical at 100 columns, want side-by-side")
	}
}

func TestLayoutStacksPanesAtMediumWidth(t *testing.T) {
	layout := layoutFor(90, 24)

	if layout.vertical {
		t.Fatal("layout is vertical, want side-by-side")
	}
}

func TestFooterTextIsClippedToTerminalWidth(t *testing.T) {
	footer := footerFor(model{width: 20, status: "12345678901234567890"})

	if len(footer) > 20 {
		t.Fatalf("footer length = %d, want <= 20: %q", len(footer), footer)
	}
}

func TestFooterOnlyShowsQuitAndHelpKeys(t *testing.T) {
	footer := footerFor(model{width: 80, status: "4 tickets"})

	for _, text := range []string{"j/k move", "enter inspect", "tab section", "? help"} {
		if !strings.Contains(footer, text) {
			t.Fatalf("footer does not show expected core hints %q: %q", text, footer)
		}
	}
}

func TestViewShowsDashboardHeaderCounts(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-ready", Title: "Ready ticket", Status: "open", Priority: 1},
		{ID: "tic-work", Title: "Started ticket", Status: "in_progress", Priority: 1},
		{ID: "tic-block", Title: "Blocked ticket", Status: "open", Priority: 2, Deps: []string{"tic-work"}},
		{ID: "tic-done", Title: "Closed ticket", Status: "closed", Priority: 3},
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.allTickets = tickets
	m.tickets = tickets
	m.detail.SetContent("Selected detail")

	view := stripANSI(m.View().Content)

	for _, want := range []string{"tk · tickets", "Ready 1", "Active 1", "Blocked 1", "Closed 1"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "mode: active") {
		t.Fatalf("view still shows legacy mode header:\n%s", view)
	}
}

func TestMainFooterShowsPrimaryActionsInPriorityOrder(t *testing.T) {
	tickets := []Ticket{{ID: "tic-ready", Status: "open", Priority: 1}}
	footer := footerFor(model{width: 120, status: "4 tickets", allTickets: tickets, tickets: tickets})

	for _, want := range []string{"j/k move", "enter inspect", "tab section", "n new", "d deps", "/ filter", "? help"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer missing %q: %q", want, footer)
		}
	}
}

func TestNarrowFooterDropsLowerPriorityHintsBeforeCoreHints(t *testing.T) {
	tickets := []Ticket{{ID: "tic-ready", Status: "open", Priority: 1}}
	footer := footerFor(model{width: 80, status: "16 tickets", allTickets: tickets, tickets: tickets})

	for _, want := range []string{"j/k move", "enter inspect", "tab section", "? help"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("narrow footer missing %q: %q", want, footer)
		}
	}
	if strings.Contains(footer, "ctrl+p") {
		t.Fatalf("narrow footer kept low-priority hint: %q", footer)
	}
}

func TestBoardListShowsProjectStatusSections(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-ready", Title: "Ready ticket", Status: "open", Priority: 1},
		{ID: "tic-work", Title: "Started ticket", Status: "in_progress", Priority: 1},
		{ID: "tic-block", Title: "Blocked ticket", Status: "open", Priority: 2, Deps: []string{"tic-work"}},
		{ID: "tic-done", Title: "Closed ticket", Status: "closed", Priority: 3},
	}
	m := model{allTickets: tickets, tickets: tickets, selected: 0, mode: ModeActive}

	view := m.renderList(44, 18)

	for _, text := range []string{"Ready", "In Progress", "Blocked", "Closed Recent", "tic-ready", "tic-work", "tic-block", "tic-done"} {
		if !strings.Contains(view, text) {
			t.Fatalf("board list missing %q:\n%s", text, view)
		}
	}
}

func TestFooterShowsContextualMovementAndActions(t *testing.T) {
	footer := footerFor(model{width: 120, status: "4 tickets", tickets: []Ticket{{ID: "tic-one"}}})

	for _, text := range []string{"j/k move", "tab section", "enter inspect", "ctrl+p cmds", "? help"} {
		if !strings.Contains(footer, text) {
			t.Fatalf("footer missing %q: %q", text, footer)
		}
	}
}

func TestMainFooterCondensesOnNarrowScreens(t *testing.T) {
	tickets := []Ticket{{ID: "tic-one", Status: "open", Priority: 1}}
	footer := footerFor(model{width: 80, status: "16 tickets", allTickets: tickets, tickets: tickets})

	for _, text := range []string{"j/k move", "enter inspect", "tab section", "? help"} {
		if !strings.Contains(footer, text) {
			t.Fatalf("narrow main footer missing %q: %q", text, footer)
		}
	}
	if len(footer) > 80 {
		t.Fatalf("footer length = %d, want <= 80: %q", len(footer), footer)
	}
}

func TestRenderTicketRowPrioritizesTitleBeforeIDAndStatus(t *testing.T) {
	tickets := []Ticket{{ID: "tic-ready", Title: "Add dashboard header", Status: "open", Priority: 1}}
	m := model{allTickets: tickets, tickets: tickets}

	row := stripANSI(m.renderTicketRow(tickets[0], 80))

	if !strings.Contains(row, "P1  Add dashboard header") {
		t.Fatalf("row missing title-first prefix: %q", row)
	}
	if !strings.Contains(row, "tic-ready") {
		t.Fatalf("row missing ticket id: %q", row)
	}
	if !strings.Contains(row, "ready") {
		t.Fatalf("row missing compact readiness label: %q", row)
	}
}

func TestRenderTicketRowShowsBlockedCountInline(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-dep", Title: "Dependency", Status: "open", Priority: 1},
		{ID: "tic-block", Title: "Blocked ticket", Status: "open", Priority: 2, Deps: []string{"tic-dep"}},
	}
	m := model{allTickets: tickets, tickets: tickets}

	row := stripANSI(m.renderTicketRow(tickets[1], 80))

	if !strings.Contains(row, "blocked·1") {
		t.Fatalf("blocked row missing inline blocker count: %q", row)
	}
}

func TestRenderListShowsCompactSectionHeaders(t *testing.T) {
	tickets := []Ticket{{ID: "tic-ready", Title: "Ready ticket", Status: "open", Priority: 1}}
	m := model{allTickets: tickets, tickets: tickets}

	view := stripANSI(m.renderList(40, 10))

	if !strings.Contains(view, "─ Ready (1)") {
		t.Fatalf("list missing compact section header:\n%s", view)
	}
}

func TestViewShowsBoardAndDetailTitles(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.allTickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1}}
	m.tickets = m.allTickets
	m.detail.SetContent("Selected detail")
	m.resizeDetail()

	view := stripANSI(m.View().Content)

	for _, text := range []string{"tk · tickets", "Queue", "Ticket"} {
		if !strings.Contains(view, text) {
			t.Fatalf("view missing %q:\n%s", text, view)
		}
	}
}

func TestRenderTicketRowKeepsStateSuffixVisibleWithinTightWidth(t *testing.T) {
	tickets := []Ticket{{ID: "tic-work", Title: "Refine metadata grid for dashboard", Status: "in_progress", Priority: 1}}
	m := model{allTickets: tickets, tickets: tickets}

	row := stripANSI(m.renderTicketRow(tickets[0], 40))

	if !strings.Contains(row, "active") {
		t.Fatalf("row lost state suffix inside tight width:\n%s", row)
	}
	if !strings.Contains(row, "tic-work") {
		t.Fatalf("row lost id inside tight width:\n%s", row)
	}
}

func TestRenderListAlignsIDAndStateColumns(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-one", Title: "Short", Status: "open", Priority: 1},
		{ID: "tic-two", Title: "Much longer ticket title", Status: "in_progress", Priority: 2},
	}
	m := model{allTickets: tickets, tickets: tickets}

	lines := strings.Split(stripANSI(m.renderList(44, 10)), "\n")
	var rowLines []string
	for _, line := range lines {
		if strings.Contains(line, "tic-one") || strings.Contains(line, "tic-two") {
			rowLines = append(rowLines, line)
		}
	}
	if len(rowLines) != 2 {
		t.Fatalf("expected 2 row lines, got %d:\n%s", len(rowLines), strings.Join(lines, "\n"))
	}
	idColOne := strings.Index(rowLines[0], "tic-one")
	idColTwo := strings.Index(rowLines[1], "tic-two")
	if idColOne != idColTwo {
		t.Fatalf("id columns are misaligned:\n%s\n%s", rowLines[0], rowLines[1])
	}
}

func TestPreviewFocusChangesDetailTitle(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.focus = focusPreview
	m.allTickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1}}
	m.tickets = m.allTickets
	m.detail.SetContent("Selected detail")

	view := stripANSI(m.View().Content)

	if !strings.Contains(view, "Preview: tic-one") {
		t.Fatalf("focused preview title missing:\n%s", view)
	}
	if strings.Contains(view, "Detail: tic-one") {
		t.Fatalf("focused preview still uses passive detail title:\n%s", view)
	}
}

func TestPreviewFooterPrioritizesEscapeOnNarrowScreens(t *testing.T) {
	footer := footerFor(model{width: 32, focus: focusPreview})

	for _, text := range []string{"preview", "esc list", "j/k scroll"} {
		if !strings.Contains(footer, text) {
			t.Fatalf("narrow preview footer missing %q: %q", text, footer)
		}
	}
	if len(footer) > 32 {
		t.Fatalf("footer length = %d, want <= 32: %q", len(footer), footer)
	}
}

func TestTabMovesSelectionToNextVisibleSection(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-ready", Title: "Ready ticket", Status: "open", Priority: 1},
		{ID: "tic-work", Title: "Started ticket", Status: "in_progress", Priority: 1},
		{ID: "tic-done", Title: "Closed ticket", Status: "closed", Priority: 2},
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.allTickets = tickets
	m.tickets = tickets
	m.selected = 0

	updated, _ := m.Update(keyMsg("tab"))

	got := selectedID(updated.(model))
	if got != "tic-work" {
		t.Fatalf("selectedID after tab = %q, want tic-work", got)
	}
}

func TestDownMovesThroughRenderedSectionOrder(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-ready", Title: "Ready ticket", Status: "open", Priority: 1},
		{ID: "tic-done", Title: "Closed ticket", Status: "closed", Priority: 2},
		{ID: "tic-work", Title: "Started ticket", Status: "in_progress", Priority: 1},
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.allTickets = tickets
	m.tickets = tickets
	m.selected = 0

	updated, _ := m.Update(keyMsg("j"))

	got := selectedID(updated.(model))
	if got != "tic-work" {
		t.Fatalf("selectedID after down = %q, want tic-work", got)
	}
}

func TestViewStacksDashboardAtEightyColumns(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 80
	m.height = 24
	m.allTickets = []Ticket{{ID: "tic-ready", Title: "Ready ticket", Status: "open", Priority: 1}}
	m.tickets = m.allTickets
	m.detail.SetContent("Status   Priority\nopen     P1")

	view := stripANSI(m.View().Content)
	queueIndex := strings.Index(view, "Queue")
	ticketIndex := strings.Index(view, "Ticket")
	if queueIndex == -1 || ticketIndex == -1 || queueIndex > ticketIndex {
		t.Fatalf("stacked layout missing queue-before-ticket order:\n%s", view)
	}
}

func TestFooterShowsSectionPositionInsteadOfOpaqueModeCycle(t *testing.T) {
	m := model{
		width:      120,
		status:     "3 tickets",
		allTickets: []Ticket{{ID: "tic-ready", Status: "open"}, {ID: "tic-work", Status: "in_progress"}},
		tickets:    []Ticket{{ID: "tic-ready", Status: "open"}, {ID: "tic-work", Status: "in_progress"}},
		selected:   0,
	}

	footer := footerFor(m)

	for _, text := range []string{"j/k move", "tab section", "enter inspect"} {
		if !strings.Contains(footer, text) {
			t.Fatalf("footer missing %q: %q", text, footer)
		}
	}
}

func TestPreviewScrollKeepsPinnedMetadataVisible(t *testing.T) {
	runner := func(name string, args ...string) (string, error) {
		return `---
status: open
priority: 1
assignee: gkarolyi
created: 2026-05-28T14:45:56Z
---
# dashboard header

line one
line two
line three
line four
line five
line six
`, nil
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, runner)
	m.width = 80
	m.height = 24
	m.allTickets = []Ticket{{ID: "tic-one", Title: "dashboard header", Status: "open", Priority: 1}}
	m.tickets = m.allTickets
	m.selected = 0
	m.resizeDetail()
	m.applyDetail(m.detailFor("tic-one"))
	m.focus = focusPreview

	updated, _ := m.Update(keyMsg("j"))
	view := stripANSI(updated.(model).View().Content)

	for _, want := range []string{"Status", "Priority", "Assignee", "Created"} {
		if !strings.Contains(view, want) {
			t.Fatalf("pinned metadata missing %q after scroll:\n%s", want, view)
		}
	}
}

func TestDetailForKeepsMetadataAboveLongMarkdown(t *testing.T) {
	runner := func(name string, args ...string) (string, error) {
		return `---
status: in_progress
priority: 1
assignee: gkarolyi
---
# Selected ticket

alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu
`, nil
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, runner)
	m.width = 100
	m.height = 28
	m.resizeDetail()

	rendered := m.detailFor("tic-one")
	detail := stripANSI(strings.TrimSpace(rendered.Header + "\n" + rendered.Body))
	if strings.Index(detail, "Status") > strings.Index(detail, "alpha beta") {
		t.Fatalf("metadata appears below markdown body:\n%s", detail)
	}
}

func TestViewSeparatesLongListRowsFromDetailPane(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.allTickets = []Ticket{
		{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1},
		{ID: "tic-two", Title: "split off ticket-tui from main ticket repo", Status: "open", Priority: 1},
	}
	m.tickets = m.allTickets
	m.detail.SetContent("Metadata\nId: tic-one")

	view := stripANSI(m.View().Content)

	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Id: tic-one") && !strings.Contains(line, "  Id: tic-one") {
			t.Fatalf("detail pane lacks a visible gutter after long list row:\n%s", view)
		}
	}
}

func TestEnterFocusesPreviewPane(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.tickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1}}
	m.detail.SetContent("Selected detail")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if updated.(model).focus != focusPreview {
		t.Fatalf("focus = %v, want focusPreview", updated.(model).focus)
	}
}

func TestEscapeReturnsFocusToTicketList(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.focus = focusPreview

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))

	if updated.(model).focus != focusTickets {
		t.Fatalf("focus = %v, want focusTickets", updated.(model).focus)
	}
}

func TestDownScrollsPreviewWhenPreviewIsFocused(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 120
	m.height = 30
	m.focus = focusPreview
	m.tickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1}}
	m.detail.SetHeight(3)
	m.detail.SetContent(strings.Repeat("line\n", 20))
	m.detail.GotoTop()

	updated, _ := m.Update(keyMsg("j"))
	updatedModel := updated.(model)

	if updatedModel.detail.YOffset() == 0 {
		t.Fatal("detail YOffset stayed at 0, want preview scroll")
	}
}

func TestDetailForWrapsTicketBodyToPreviewWidth(t *testing.T) {
	runner := func(name string, args ...string) (string, error) {
		return `---
id: tic-one
---
# Selected ticket

alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron
`, nil
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, runner)
	m.width = 60
	m.height = 24
	m.resizeDetail()

	rendered := m.detailFor("tic-one")
	detail := stripANSI(strings.TrimSpace(rendered.Header + "\n" + rendered.Body))

	if strings.Contains(detail, "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron") {
		t.Fatalf("detail contains unwrapped long line:\n%s", detail)
	}
}

func TestLoadCmdLoadsDetailForFirstRenderedTicket(t *testing.T) {
	dir := t.TempDir()
	writeTestTicket(t, dir, "tic-work", "in_progress", 1, "Started ticket")
	writeTestTicket(t, dir, "tic-ready", "open", 2, "Ready ticket")
	runner := func(name string, args ...string) (string, error) {
		id := args[len(args)-1]
		return "---\nid: " + id + "\n---\n# " + id, nil
	}
	m := newModel(Config{TicketsDir: dir, TKScript: "/usr/local/bin/tk"}, runner)
	m.width = 120
	m.height = 30

	msg := m.loadCmd()().(loadedMsg)

	if !strings.Contains(msg.detail.Header, "tic-ready") {
		t.Fatalf("loaded detail does not match first rendered ready ticket:\n%#v", msg.detail)
	}
}

func TestLoadCmdPreservesSelectedTicketDetailOnRefresh(t *testing.T) {
	dir := t.TempDir()
	writeTestTicket(t, dir, "tic-work", "in_progress", 1, "Started ticket")
	writeTestTicket(t, dir, "tic-ready", "open", 2, "Ready ticket")
	runner := func(name string, args ...string) (string, error) {
		id := args[len(args)-1]
		return "---\nid: " + id + "\n---\n# " + id, nil
	}
	m := newModel(Config{TicketsDir: dir, TKScript: "/usr/local/bin/tk"}, runner)
	m.width = 120
	m.height = 30
	m.tickets = []Ticket{
		{ID: "tic-ready", Status: "open", Priority: 2},
		{ID: "tic-work", Status: "in_progress", Priority: 1},
	}
	m.selected = 1

	msg := m.loadCmd()().(loadedMsg)

	if !strings.Contains(msg.detail.Header, "tic-work") {
		t.Fatalf("loaded detail does not match selected ticket after refresh:\n%#v", msg.detail)
	}
}

func TestWindowResizeReloadsDetailForCurrentWidth(t *testing.T) {
	runner := func(name string, args ...string) (string, error) {
		return `---
status: open
priority: 1
assignee: gkarolyi
tags: [ui, ux]
created: 2026-05-28
---
# dashboard header

Body text.
`, nil
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, runner)
	m.allTickets = []Ticket{{ID: "tic-ready", Title: "dashboard header", Status: "open", Priority: 1}}
	m.tickets = m.allTickets
	m.selected = 0

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd == nil {
		t.Fatal("window resize did not request detail reload")
	}
	msg := cmd().(detailLoadedMsg)

	detail := stripANSI(strings.TrimSpace(msg.detail.Header + "\n" + msg.detail.Body))
	if !strings.Contains(detail, "Status") || !strings.Contains(detail, "Priority") {
		t.Fatalf("resized detail was not re-rendered for current width:\n%s", detail)
	}
}

func dashboardFixtureModel(width int, height int) model {
	tickets := []Ticket{
		{ID: "tic-ready", Title: "Add dashboard header", Status: "open", Priority: 1},
		{ID: "tic-work", Title: "Refine metadata grid", Status: "in_progress", Priority: 1},
		{ID: "tic-block", Title: "Blocked by dependency", Status: "open", Priority: 2, Deps: []string{"tic-work"}},
		{ID: "tic-done", Title: "Closed example", Status: "closed", Priority: 3},
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = width
	m.height = height
	m.allTickets = tickets
	m.tickets = tickets
	m.detail.SetContent("Status   Priority\nin_progress  P1\n\nSummary\nRefine metadata grid")
	m.resizeDetail()
	return m
}

func TestDashboardFixtureViewAt120x40(t *testing.T) {
	view := stripANSI(dashboardFixtureModel(120, 40).View().Content)
	for _, want := range []string{"tk · tickets", "Ready 1", "Active 1", "Blocked 1", "Closed 1", "Queue", "Ticket"} {
		if !strings.Contains(view, want) {
			t.Fatalf("120x40 view missing %q:\n%s", want, view)
		}
	}
}

func TestDashboardFixtureViewAt80x24(t *testing.T) {
	view := stripANSI(dashboardFixtureModel(80, 24).View().Content)
	if strings.Index(view, "Queue") > strings.Index(view, "Ticket") {
		t.Fatalf("80x24 view does not stack queue before ticket:\n%s", view)
	}
}

func TestDashboardFixturePreviewAt80x24(t *testing.T) {
	m := dashboardFixtureModel(80, 24)
	m.focus = focusPreview
	view := stripANSI(m.View().Content)
	for _, want := range []string{"Preview:", "preview", "esc list", "j/k scroll"} {
		if !strings.Contains(view, want) {
			t.Fatalf("80x24 preview missing %q:\n%s", want, view)
		}
	}
}

func writeTestTicket(t *testing.T, dir string, id string, status string, priority int, title string) {
	t.Helper()
	content := fmt.Sprintf("---\nid: %s\nstatus: %s\ndeps: []\nlinks: []\npriority: %d\n---\n# %s\n", id, status, priority, title)
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stripANSI(value string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(value, "")
}

func TestHelpKeyShowsComprehensiveHelpModal(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 30

	updated, _ := m.Update(keyMsg("?"))
	view := updated.(model).View().Content

	for _, text := range []string{"Help", "j/down", "tab", "n", "/", "e", "s", "c", "r", "R", "q", "?"} {
		if !strings.Contains(view, text) {
			t.Fatalf("help view missing %q:\n%s", text, view)
		}
	}
}

func TestEscapeClosesHelpModal(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 30

	updated, _ := m.Update(keyMsg("?"))
	updated, _ = updated.(model).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))

	if updated.(model).helpShown {
		t.Fatal("helpShown is true, want false")
	}
}

func TestCommandPaletteKeyShowsSearchableCommands(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 30
	m.tickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 2}}

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	view := updated.(model).View().Content

	for _, text := range []string{"Command Palette", "create ticket", "query tickets", "close selected ticket"} {
		if !strings.Contains(view, text) {
			t.Fatalf("palette view missing %q:\n%s", text, view)
		}
	}
	if strings.Contains(view, "move selection") {
		t.Fatalf("palette includes movement commands:\n%s", view)
	}
}

func TestHelpOverlayKeepsDashboardVisibleBehindModal(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 28
	m.tickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1}}
	m.allTickets = m.tickets
	m.detail.SetContent("Status    Priority\nopen      1\n\nSelected detail")
	m.resizeDetail()

	updated, _ := m.Update(keyMsg("?"))
	view := stripANSI(updated.(model).View().Content)

	for _, want := range []string{"Help", "Queue", "Status"} {
		if !strings.Contains(view, want) {
			t.Fatalf("overlay view missing %q:\n%s", want, view)
		}
	}
}

func TestPaletteOverlayKeepsDashboardVisibleBehindModal(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 28
	m.tickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 1}}
	m.allTickets = m.tickets
	m.detail.SetContent("Status    Priority\nopen      1\n\nSelected detail")
	m.resizeDetail()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	view := stripANSI(updated.(model).View().Content)

	for _, want := range []string{"Command Palette", "Queue", "Status"} {
		if !strings.Contains(view, want) {
			t.Fatalf("palette overlay missing %q:\n%s", want, view)
		}
	}
}

func TestHelpKeyKeepsSelectedTicketVisible(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 30
	m.tickets = []Ticket{{ID: "tic-one", Title: "Selected ticket", Status: "open", Priority: 2}}

	updated, _ := m.Update(keyMsg("?"))
	view := updated.(model).View().Content

	for _, text := range []string{"Help", "Queue"} {
		if !strings.Contains(view, text) {
			t.Fatalf("help view missing %q:\n%s", text, view)
		}
	}
}

func TestCommandPaletteFuzzyFiltersCommands(t *testing.T) {
	commands := filterPaletteCommands("crt", paletteCommands())

	if len(commands) != 1 {
		t.Fatalf("len(commands) = %d, want 1: %#v", len(commands), commands)
	}
	if commands[0].name != "create ticket" {
		t.Fatalf("command = %q, want create ticket", commands[0].name)
	}
}

func TestCommandPaletteEnterExecutesSelectedCommand(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 30

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	updated, _ = updated.(model).Update(keyMsg("c"))
	updated, _ = updated.(model).Update(keyMsg("r"))
	updated, _ = updated.(model).Update(keyMsg("t"))
	updated, _ = updated.(model).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if updated.(model).prompt != promptCreate {
		t.Fatalf("prompt = %v, want promptCreate", updated.(model).prompt)
	}
}

func TestPaletteWindowKeepsFixedRowCountNearEnd(t *testing.T) {
	commands := []paletteCommand{
		{name: "one", key: "1"},
		{name: "two", key: "2"},
		{name: "three", key: "3"},
	}
	rows := paletteRows(commands, 2, 5)

	if len(rows) != 5 {
		t.Fatalf("len(rows) = %d, want 5", len(rows))
	}
	if rows[3] != "" || rows[4] != "" {
		t.Fatalf("last rows = %#v, want padded empty rows", rows[3:])
	}
}

func TestInitLoadsTicketsAndStartsPeriodicRefresh(t *testing.T) {
	cmd := newModel(Config{}, nil).Init()

	if cmd == nil {
		t.Fatal("Init returned nil, want batched load and refresh commands")
	}
}

func TestRefreshTickReloadsTickets(t *testing.T) {
	m := newModel(Config{}, nil)

	_, cmd := m.Update(refreshTickMsg{})

	if cmd == nil {
		t.Fatal("refresh tick returned nil command, want reload command")
	}
}

func TestDependencyKeyShowsTicketPicker(t *testing.T) {
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, nil)
	m.width = 100
	m.height = 30
	m.tickets = []Ticket{
		{ID: "tic-one", Title: "Current ticket", Status: "open", Priority: 2},
		{ID: "tic-two", Title: "Dependency ticket", Status: "open", Priority: 2},
	}
	m.allTickets = m.tickets

	updated, _ := m.Update(keyMsg("d"))
	view := updated.(model).View().Content

	if !strings.Contains(view, "Add Dependencies") {
		t.Fatalf("view does not contain dependency modal title:\n%s", view)
	}
	if !strings.Contains(view, "tic-two") || !strings.Contains(view, "Dependency ticket") {
		t.Fatalf("view does not contain dependency candidate:\n%s", view)
	}
}

func TestDependencyPickerRunsDepForSelectedTickets(t *testing.T) {
	var calls [][]string
	runner := func(name string, args ...string) (string, error) {
		calls = append(calls, append([]string{name}, args...))
		return "added dependency", nil
	}
	m := newModel(Config{TKScript: "/usr/local/bin/tk"}, runner)
	m.width = 100
	m.height = 30
	m.tickets = []Ticket{
		{ID: "tic-one", Title: "Current ticket", Status: "open", Priority: 2},
		{ID: "tic-two", Title: "Dependency ticket", Status: "open", Priority: 2},
	}
	m.allTickets = m.tickets

	updated, _ := m.Update(keyMsg("d"))
	updated, _ = updated.(model).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Text: " "}))
	_, cmd := updated.(model).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	msg := cmd()

	if _, ok := msg.(statusMsg); !ok {
		t.Fatalf("msg = %#v, want statusMsg", msg)
	}
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1: %#v", len(calls), calls)
	}
	assertStrings(t, calls[0], []string{"/usr/local/bin/tk", "super", "dep", "tic-one", "tic-two"})
}

func keyMsg(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: []rune(value)[0], Text: value})
}
