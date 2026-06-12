package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTicketFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc-1234.md")
	content := `---
id: abc-1234
status: open
deps: [abc-0001, abc-0002]
links: []
created: 2026-05-23T00:00:00Z
type: task
priority: 1
assignee: Ada Lovelace
tags: [ui, backend]
---
# Build TUI

Details here.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ticket, err := ParseTicketFile(path)
	if err != nil {
		t.Fatalf("ParseTicketFile returned error: %v", err)
	}

	if ticket.ID != "abc-1234" {
		t.Fatalf("ID = %q", ticket.ID)
	}
	if ticket.Status != "open" {
		t.Fatalf("Status = %q", ticket.Status)
	}
	if ticket.Priority != 1 {
		t.Fatalf("Priority = %d", ticket.Priority)
	}
	if ticket.Title != "Build TUI" {
		t.Fatalf("Title = %q", ticket.Title)
	}
	if ticket.Assignee != "Ada Lovelace" {
		t.Fatalf("Assignee = %q", ticket.Assignee)
	}
	assertStrings(t, ticket.Deps, []string{"abc-0001", "abc-0002"})
	assertStrings(t, ticket.Tags, []string{"ui", "backend"})
}

func TestParseTicketFileDefaultsPriorityAndTitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc-1234.md")
	content := `---
id: abc-1234
status: in_progress
deps: []
---
Body without heading.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ticket, err := ParseTicketFile(path)
	if err != nil {
		t.Fatalf("ParseTicketFile returned error: %v", err)
	}

	if ticket.Priority != 2 {
		t.Fatalf("Priority = %d", ticket.Priority)
	}
	if ticket.Title != "(untitled)" {
		t.Fatalf("Title = %q", ticket.Title)
	}
}

func TestFilterTicketsClassifiesReadyBlockedAndClosed(t *testing.T) {
	tickets := []Ticket{
		{ID: "a-0001", Status: "open", Priority: 2, Title: "No deps"},
		{ID: "a-0002", Status: "open", Priority: 1, Title: "Closed dep", Deps: []string{"a-0004"}},
		{ID: "a-0003", Status: "open", Priority: 3, Title: "Open dep", Deps: []string{"a-0001"}},
		{ID: "a-0004", Status: "closed", Priority: 0, Title: "Done"},
		{ID: "a-0005", Status: "open", Priority: 2, Title: "Missing dep", Deps: []string{"missing"}},
	}

	ready := FilterTickets(tickets, ModeReady)
	assertIDs(t, ready, []string{"a-0002", "a-0001"})

	blocked := FilterTickets(tickets, ModeBlocked)
	assertIDs(t, blocked, []string{"a-0005", "a-0003"})

	closed := FilterTickets(tickets, ModeClosed)
	assertIDs(t, closed, []string{"a-0004"})
}

func TestFilterTicketsActiveShowsClosedAfterActiveTickets(t *testing.T) {
	tickets := []Ticket{
		{ID: "a-0003", Status: "closed", Priority: 0, Title: "Closed high"},
		{ID: "a-0002", Status: "in_progress", Priority: 3, Title: "Doing"},
		{ID: "a-0001", Status: "open", Priority: 2, Title: "Open"},
		{ID: "a-0004", Status: "closed", Priority: 4, Title: "Closed low"},
	}

	active := FilterTickets(tickets, ModeActive)
	assertIDs(t, active, []string{"a-0001", "a-0002", "a-0003", "a-0004"})
}

func TestFilterTicketSearchMatchesIDTitleAndDescription(t *testing.T) {
	tickets := []Ticket{
		{ID: "alpha-1", Title: "Add dashboard header", Description: "Improve the top summary"},
		{ID: "beta-2", Title: "Create dependency modal", Description: "Pick related tickets"},
		{ID: "gamma-3", Title: "Markdown preview", Description: "Render long ticket descriptions"},
	}

	assertIDs(t, FilterTicketSearch(tickets, "b2"), []string{"beta-2"})
	assertIDs(t, FilterTicketSearch(tickets, "dash"), []string{"alpha-1"})
	assertIDs(t, FilterTicketSearch(tickets, "long desc"), []string{"gamma-3"})
}

func TestFilterTicketSearchDoesNotMatchScatteredLettersAcrossFullBody(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-ready", Title: "dashboard header", Description: "Design a clearer dashboard-first UX for ticket-tui. Acceptance header counts visible queue remains readable"},
		{ID: "tic-work", Title: "metadata grid", Description: "Keep metadata compact and scannable"},
	}

	assertIDs(t, FilterTicketSearch(tickets, "scann"), []string{"tic-work"})
}

func TestFilterTicketSearchEmptyQueryReturnsTickets(t *testing.T) {
	tickets := []Ticket{
		{ID: "tic-alpha", Title: "Add dashboard header"},
		{ID: "tic-beta", Title: "Create dependency modal"},
	}

	assertIDs(t, FilterTicketSearch(tickets, "  "), []string{"tic-alpha", "tic-beta"})
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func assertIDs(t *testing.T, got []Ticket, want []string) {
	t.Helper()
	ids := make([]string, len(got))
	for i := range got {
		ids[i] = got[i].ID
	}
	assertStrings(t, ids, want)
}
