package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Mode int

const (
	ModeActive Mode = iota
	ModeReady
	ModeBlocked
	ModeClosed
)

var modeNames = []string{"active", "ready", "blocked", "closed"}

type Ticket struct {
	ID       string
	Status   string
	Priority int
	Title    string
	Deps     []string
	Assignee string
	Tags     []string
	Path     string
}

func ParseTicketFile(path string) (Ticket, error) {
	file, err := os.Open(path)
	if err != nil {
		return Ticket{}, err
	}
	defer file.Close()

	ticket := Ticket{Priority: 2, Title: "(untitled)", Path: path}
	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	seenFrontmatter := false
	frontmatterDone := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" && !frontmatterDone {
			if !seenFrontmatter {
				seenFrontmatter = true
				inFrontmatter = true
			} else {
				inFrontmatter = false
				frontmatterDone = true
			}
			continue
		}

		if inFrontmatter {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			switch key {
			case "id":
				ticket.ID = value
			case "status":
				ticket.Status = value
			case "priority":
				if priority, err := strconv.Atoi(value); err == nil {
					ticket.Priority = priority
				}
			case "deps":
				ticket.Deps = parseInlineList(value)
			case "assignee":
				ticket.Assignee = value
			case "tags":
				ticket.Tags = parseInlineList(value)
			}
			continue
		}

		if frontmatterDone && strings.HasPrefix(line, "# ") && ticket.Title == "(untitled)" {
			ticket.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}

	if err := scanner.Err(); err != nil {
		return Ticket{}, err
	}
	if ticket.ID == "" {
		ticket.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return ticket, nil
}

func LoadTickets(dir string) ([]Ticket, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}

	tickets := make([]Ticket, 0, len(matches))
	for _, path := range matches {
		ticket, err := ParseTicketFile(path)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	SortTickets(tickets)
	return tickets, nil
}

func FilterTickets(tickets []Ticket, mode Mode) []Ticket {
	byID := make(map[string]Ticket, len(tickets))
	for _, ticket := range tickets {
		byID[ticket.ID] = ticket
	}

	filtered := make([]Ticket, 0, len(tickets))
	for _, ticket := range tickets {
		switch mode {
		case ModeActive:
			if isActive(ticket) || ticket.Status == "closed" {
				filtered = append(filtered, ticket)
			}
		case ModeReady:
			if isActive(ticket) && isReady(ticket, byID) {
				filtered = append(filtered, ticket)
			}
		case ModeBlocked:
			if isActive(ticket) && isBlocked(ticket, byID) {
				filtered = append(filtered, ticket)
			}
		case ModeClosed:
			if ticket.Status == "closed" {
				filtered = append(filtered, ticket)
			}
		}
	}
	SortTickets(filtered)
	return filtered
}

func SortTickets(tickets []Ticket) {
	sort.Slice(tickets, func(i, j int) bool {
		if activeRank(tickets[i]) != activeRank(tickets[j]) {
			return activeRank(tickets[i]) < activeRank(tickets[j])
		}
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].ID < tickets[j].ID
	})
}

func activeRank(ticket Ticket) int {
	if ticket.Status == "closed" {
		return 1
	}
	return 0
}

func ModeName(mode Mode) string {
	if int(mode) < 0 || int(mode) >= len(modeNames) {
		return "active"
	}
	return modeNames[mode]
}

func NextMode(mode Mode) Mode {
	return Mode((int(mode) + 1) % len(modeNames))
}

func parseInlineList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func isActive(ticket Ticket) bool {
	return ticket.Status == "open" || ticket.Status == "in_progress"
}

func isReady(ticket Ticket, byID map[string]Ticket) bool {
	for _, dep := range ticket.Deps {
		depTicket, ok := byID[dep]
		if !ok || depTicket.Status != "closed" {
			return false
		}
	}
	return true
}

func isBlocked(ticket Ticket, byID map[string]Ticket) bool {
	for _, dep := range ticket.Deps {
		depTicket, ok := byID[dep]
		if !ok || depTicket.Status != "closed" {
			return true
		}
	}
	return false
}
