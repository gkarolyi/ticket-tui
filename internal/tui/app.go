package tui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Config struct {
	TicketsDir string
	TKScript   string
}

type commandRunner func(name string, args ...string) (string, error)

type commandSpecValue struct {
	name string
	args []string
}

type model struct {
	config     Config
	runner     commandRunner
	width      int
	height     int
	focus      focusArea
	allTickets []Ticket
	tickets    []Ticket
	selected   int
	mode       Mode
	detail     viewport.Model
	status     string
	err        string
	prompt     promptKind
	input      textinput.Model
	queryShown bool
	helpShown  bool
	palette    paletteState
	depPicker  dependencyPickerState
}

type paletteState struct {
	shown    bool
	input    textinput.Model
	selected int
}

type paletteCommand struct {
	name string
	key  string
}

type dependencyPickerState struct {
	shown    bool
	selected int
	marked   map[string]bool
}

type focusArea int

const (
	focusTickets focusArea = iota
	focusPreview
)

type promptKind int

const (
	promptNone promptKind = iota
	promptCreate
	promptQuery
)

type loadedMsg struct {
	tickets []Ticket
	detail  string
}

type statusMsg struct {
	text string
}

type errorMsg struct {
	err error
}

type refreshTickMsg struct{}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("63"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	modalStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Width(52)
)

const (
	narrowLayoutWidth = 100
	refreshInterval   = 5 * time.Second
)

type layoutSpec struct {
	vertical     bool
	bodyHeight   int
	listWidth    int
	gutterWidth  int
	detailWidth  int
	listHeight   int
	detailHeight int
}

func NewConfig(env map[string]string) (Config, error) {
	ticketsDir := env["TICKETS_DIR"]
	if ticketsDir == "" {
		startDir := env["PWD"]
		if startDir == "" {
			var err error
			startDir, err = os.Getwd()
			if err != nil {
				return Config{}, err
			}
		}

		foundDir, err := findTicketsDir(startDir)
		if err != nil {
			return Config{}, errors.New("no .tickets directory found; run 'tk create' to initialize, or set TICKETS_DIR")
		}
		ticketsDir = foundDir
	}
	return Config{TicketsDir: ticketsDir, TKScript: env["TK_SCRIPT"]}, nil
}

func findTicketsDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".tickets")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

func Run(config Config) error {
	program := tea.NewProgram(
		newModel(config, defaultRunner),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)
	_, err := program.Run()
	return err
}

func newModel(config Config, runner commandRunner) model {
	detail := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	detail.SetContent("Loading...")
	return model{
		config: config,
		runner: runner,
		mode:   ModeActive,
		detail: detail,
		status: "Loading tickets...",
		input:  newTextInput(""),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), refreshTickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeDetail()
		return m, nil
	case loadedMsg:
		m.allTickets = msg.tickets
		m.tickets = FilterTickets(m.allTickets, m.mode)
		m.tickets = m.orderedTickets()
		m.queryShown = false
		if m.selected >= len(m.tickets) {
			m.selected = len(m.tickets) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		m.detail.SetContent(msg.detail)
		m.status = fmt.Sprintf("%d tickets", len(m.tickets))
		m.err = ""
		return m, nil
	case statusMsg:
		m.status = msg.text
		m.err = ""
		return m, m.loadCmd()
	case errorMsg:
		m.err = msg.err.Error()
		return m, nil
	case queryMsg:
		m.detail.SetContent(msg.output)
		m.status = "Query: " + msg.filter
		m.err = ""
		m.queryShown = true
		return m, nil
	case refreshTickMsg:
		return m, tea.Batch(m.loadCmd(), refreshTickCmd())
	case tea.KeyPressMsg:
		if m.prompt != promptNone {
			return m.updatePrompt(msg)
		}
		if m.depPicker.shown {
			return m.updateDependencyPicker(msg)
		}
		if m.palette.shown {
			return m.updatePalette(msg)
		}
		switch msg.String() {
		case "esc":
			if m.helpShown {
				m.helpShown = false
				m.status = "Closed help"
				return m, nil
			}
			if m.queryShown {
				m.queryShown = false
				m.status = "Returned to ticket detail"
				return m, m.loadDetailCmd()
			}
			if m.focus == focusPreview {
				m.focus = focusTickets
				m.status = "Focused ticket list"
				return m, nil
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "down":
			if m.focus == focusPreview {
				m.detail.ScrollDown(1)
				return m, nil
			}
			m.tickets = m.orderedTickets()
			if m.selected < len(m.tickets)-1 {
				m.selected++
				return m, m.loadDetailCmd()
			}
		case "k", "up":
			if m.focus == focusPreview {
				m.detail.ScrollUp(1)
				return m, nil
			}
			m.tickets = m.orderedTickets()
			if m.selected > 0 {
				m.selected--
				return m, m.loadDetailCmd()
			}
		case "enter":
			if selectedID(m) != "" {
				m.focus = focusPreview
				m.status = "Focused preview"
			}
			return m, nil
		case "tab":
			m.tickets = m.orderedTickets()
			if len(m.tickets) > 0 {
				m.selected = m.nextSectionFirstIndex()
			}
			return m, m.loadDetailCmd()
		case "R":
			m.status = "Refreshing..."
			return m, m.loadCmd()
		case "n":
			m.prompt = promptCreate
			m.input = newTextInput("Ticket title")
			m.status = "Create ticket title:"
			return m, nil
		case "/":
			m.prompt = promptQuery
			m.input = newTextInput(`.status == "open"`)
			m.status = "Query filter:"
			return m, nil
		case "?":
			m.helpShown = true
			m.status = "Help"
			return m, nil
		case "ctrl+p":
			m.palette = paletteState{shown: true, input: newPaletteInput()}
			m.status = "Command palette"
			return m, nil
		case "d":
			m.depPicker = dependencyPickerState{shown: true, marked: map[string]bool{}}
			m.status = "Add dependencies"
			return m, nil
		case "s":
			return m, m.actionCmd("start")
		case "c":
			return m, m.actionCmd("close")
		case "r":
			return m, m.actionCmd("reopen")
		case "e":
			return m, m.editCmd()
		}
	}

	var cmd tea.Cmd
	m.detail, cmd = m.detail.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	header := titleStyle.Render("tk · ticket board") + " " + mutedStyle.Render("mode: "+ModeName(m.mode))
	layout := layoutFor(m.width, m.height)

	list := lipgloss.NewStyle().Width(layout.listWidth).Height(layout.listHeight).Render(m.renderList(layout.listWidth, layout.listHeight))
	detailContent := m.detail.View()
	if m.prompt != promptNone {
		detailContent = m.renderPromptModal()
	} else if m.depPicker.shown {
		detailContent = m.renderDependencyPickerModal()
	} else if m.palette.shown {
		detailContent = m.renderPaletteModal()
	} else if m.helpShown {
		detailContent = renderHelpModal()
	}
	detailTitle := "Detail"
	if id := selectedID(m); id != "" {
		detailTitle = "Detail: " + id
		if m.focus == focusPreview {
			detailTitle = "Preview: " + id
		}
	}
	detailTitleStyle := titleStyle
	if m.focus == focusPreview {
		detailTitleStyle = selectedStyle
	}
	detail := lipgloss.NewStyle().Width(layout.detailWidth).Height(layout.detailHeight).Render(
		lipgloss.JoinVertical(lipgloss.Left, detailTitleStyle.Render(detailTitle), detailContent),
	)
	gutter := strings.Repeat(" ", layout.gutterWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top, list, gutter, detail)
	if layout.vertical {
		body = lipgloss.JoinVertical(lipgloss.Left, list, detail)
	}

	footerText := footerFor(m)
	footerStyle := mutedStyle
	if m.err != "" {
		footerStyle = errorStyle
	}

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body, footerStyle.Render(footerText)))
	view.AltScreen = true
	return view
}

func layoutFor(width, height int) layoutSpec {
	bodyHeight := max(1, height-3)
	if width <= narrowLayoutWidth {
		listHeight := max(3, bodyHeight/3+1)
		return layoutSpec{
			vertical:     true,
			bodyHeight:   bodyHeight,
			listWidth:    width,
			detailWidth:  width,
			listHeight:   listHeight,
			detailHeight: max(1, bodyHeight-listHeight),
		}
	}

	leftWidth := max(28, width/3)
	gutterWidth := 2
	rightWidth := max(20, width-leftWidth-gutterWidth)
	return layoutSpec{
		bodyHeight:   bodyHeight,
		listWidth:    leftWidth,
		gutterWidth:  gutterWidth,
		detailWidth:  rightWidth,
		listHeight:   bodyHeight,
		detailHeight: bodyHeight,
	}
}

func footerFor(m model) string {
	footerText := mainFooterFor(m)
	if m.err != "" {
		footerText = m.err
	} else if m.focus == focusPreview {
		footerText = "preview | esc list  j/k scroll  e edit"
	} else if m.prompt != promptNone {
		footerText = "enter submit  esc cancel"
	} else if m.depPicker.shown {
		footerText = "space select  enter save  esc cancel"
	} else if m.palette.shown {
		footerText = "enter run  esc cancel"
	} else if m.helpShown {
		footerText = "esc close help"
	} else if m.status != "" && m.width > narrowLayoutWidth {
		footerText = m.status + " | " + footerText
	}
	return truncateText(footerText, m.width)
}

func mainFooterFor(m model) string {
	if m.width <= narrowLayoutWidth {
		parts := make([]string, 0, 6)
		if status := compactFooterStatus(m.status); status != "" {
			parts = append(parts, status)
		}
		if section := m.currentSectionName(); section != "-" {
			parts = append(parts, section)
		}
		parts = append(parts, "q quit", "? help", "j/k move", "tab section")
		return strings.Join(parts, " | ")
	}
	return fmt.Sprintf("section %s | q quit  ? help  j/k move  tab next section  enter inspect  ctrl+p palette", m.currentSectionName())
}

func compactFooterStatus(status string) string {
	if status == "Focused ticket list" {
		return "list"
	}
	return status
}

func renderHelpModal() string {
	commands := []string{
		titleStyle.Render("Help"),
		"j/down      move selection down",
		"k/up        move selection up",
		"tab         jump to next section",
		"n           create ticket",
		"/           query tickets",
		"d           add dependencies",
		"e           edit selected ticket",
		"s           start selected ticket",
		"c           close selected ticket",
		"r           reopen selected ticket",
		"R           refresh tickets",
		"q           quit",
		"?           show this help",
		"ctrl+p      command palette",
		mutedStyle.Render("esc close"),
	}
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, commands...))
}

func paletteCommands() []paletteCommand {
	return []paletteCommand{
		{name: "create ticket", key: "n"},
		{name: "query tickets", key: "/"},
		{name: "add dependencies", key: "d"},
		{name: "edit selected ticket", key: "e"},
		{name: "start selected ticket", key: "s"},
		{name: "close selected ticket", key: "c"},
		{name: "reopen selected ticket", key: "r"},
		{name: "refresh tickets", key: "R"},
		{name: "show help", key: "?"},
		{name: "quit", key: "q"},
	}
}

func filterPaletteCommands(query string, commands []paletteCommand) []paletteCommand {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return commands
	}
	filtered := make([]paletteCommand, 0, len(commands))
	for _, command := range commands {
		if fuzzyMatch(query, strings.ToLower(command.name)) {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

func fuzzyMatch(query, value string) bool {
	position := 0
	for _, queryRune := range query {
		found := false
		for position < len(value) {
			if rune(value[position]) == queryRune {
				position++
				found = true
				break
			}
			position++
		}
		if !found {
			return false
		}
	}
	return true
}

func (m model) renderPaletteModal() string {
	commands := filterPaletteCommands(m.palette.input.Value(), paletteCommands())
	lines := []string{titleStyle.Render("Command Palette"), m.palette.input.View()}
	if len(commands) == 0 {
		lines = append(lines, mutedStyle.Render("No commands"))
	} else {
		rows := paletteRows(commands, m.palette.selected, 8)
		for i, row := range rows {
			line := row
			if i+paletteStart(len(commands), m.palette.selected, 8) == m.palette.selected && row != "" {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, mutedStyle.Render("enter run  esc cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func paletteRows(commands []paletteCommand, selected int, height int) []string {
	rows := make([]string, 0, height)
	start := paletteStart(len(commands), selected, height)
	end := min(len(commands), start+height)
	for i := start; i < end; i++ {
		rows = append(rows, fmt.Sprintf("%-10s %s", commands[i].key, commands[i].name))
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	return rows
}

func paletteStart(count int, selected int, height int) int {
	if count <= height {
		return 0
	}
	start := selected - height/2
	if start < 0 {
		return 0
	}
	if start+height > count {
		return count - height
	}
	return start
}

func (m model) updatePalette(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.palette = paletteState{}
		m.status = "Cancelled"
		return m, nil
	case "enter":
		commands := filterPaletteCommands(m.palette.input.Value(), paletteCommands())
		if len(commands) == 0 {
			m.status = "No command selected"
			return m, nil
		}
		selected := min(m.palette.selected, len(commands)-1)
		m.palette = paletteState{}
		return m.runPaletteCommand(commands[selected])
	case "down", "j":
		commands := filterPaletteCommands(m.palette.input.Value(), paletteCommands())
		if m.palette.selected < len(commands)-1 {
			m.palette.selected++
		}
		return m, nil
	case "up", "k":
		if m.palette.selected > 0 {
			m.palette.selected--
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.palette.input, cmd = m.palette.input.Update(msg)
		commands := filterPaletteCommands(m.palette.input.Value(), paletteCommands())
		if m.palette.selected >= len(commands) {
			m.palette.selected = max(0, len(commands)-1)
		}
		return m, cmd
	}
}

func (m model) runPaletteCommand(command paletteCommand) (tea.Model, tea.Cmd) {
	switch command.key {
	case "n":
		return m.Update(keyMsgFor("n"))
	case "/":
		return m.Update(keyMsgFor("/"))
	case "d":
		return m.Update(keyMsgFor("d"))
	case "e":
		return m.Update(keyMsgFor("e"))
	case "s":
		return m.Update(keyMsgFor("s"))
	case "c":
		return m.Update(keyMsgFor("c"))
	case "r":
		return m.Update(keyMsgFor("r"))
	case "R":
		return m.Update(keyMsgFor("R"))
	case "?":
		return m.Update(keyMsgFor("?"))
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m model) renderDependencyPickerModal() string {
	candidates := m.dependencyCandidates()
	lines := []string{titleStyle.Render("Add Dependencies")}
	if len(candidates) == 0 {
		lines = append(lines, mutedStyle.Render("No other tickets"))
	} else {
		start := max(0, m.depPicker.selected-6)
		end := min(len(candidates), start+8)
		for i := start; i < end; i++ {
			marker := "[ ]"
			if m.depPicker.marked[candidates[i].ID] {
				marker = "[x]"
			}
			line := fmt.Sprintf("%s %-8s %s", marker, candidates[i].ID, candidates[i].Title)
			if i == m.depPicker.selected {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, mutedStyle.Render("space select  enter save  esc cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) updateDependencyPicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.depPicker = dependencyPickerState{}
		m.status = "Cancelled"
		return m, nil
	case "down", "j":
		candidates := m.dependencyCandidates()
		if m.depPicker.selected < len(candidates)-1 {
			m.depPicker.selected++
		}
		return m, nil
	case "up", "k":
		if m.depPicker.selected > 0 {
			m.depPicker.selected--
		}
		return m, nil
	case "space":
		candidates := m.dependencyCandidates()
		if len(candidates) == 0 {
			return m, nil
		}
		id := candidates[m.depPicker.selected].ID
		m.depPicker.marked[id] = !m.depPicker.marked[id]
		return m, nil
	case "enter":
		ids := m.selectedDependencyIDs()
		m.depPicker = dependencyPickerState{}
		return m, m.addDependenciesCmd(ids)
	}
	return m, nil
}

func (m model) dependencyCandidates() []Ticket {
	currentID := selectedID(m)
	if currentID == "" {
		return nil
	}
	candidates := make([]Ticket, 0, len(m.allTickets))
	for _, ticket := range m.allTickets {
		if ticket.ID != currentID {
			candidates = append(candidates, ticket)
		}
	}
	return candidates
}

func (m model) selectedDependencyIDs() []string {
	candidates := m.dependencyCandidates()
	ids := make([]string, 0, len(candidates))
	for _, ticket := range candidates {
		if m.depPicker.marked[ticket.ID] {
			ids = append(ids, ticket.ID)
		}
	}
	return ids
}

func (m model) renderList(width, height int) string {
	if len(m.tickets) == 0 {
		return mutedStyle.Render("No tickets. Run 'tk create' to add one.")
	}

	lines := make([]string, 0, height)
	for _, section := range m.visibleSections() {
		lines = append(lines, titleStyle.Render(truncateText(fmt.Sprintf("─ %s (%d)", section.name, len(section.tickets)), width)))
		if len(lines) >= height {
			break
		}
		for _, ticket := range section.tickets {
			line := m.renderTicketRow(ticket, width)
			lines = append(lines, line)
			if len(lines) >= height {
				break
			}
		}
		if len(lines) >= height {
			break
		}
	}
	return strings.Join(lines, "\n")
}

type ticketSection struct {
	name    string
	tickets []Ticket
}

func (m model) visibleSections() []ticketSection {
	byID := ticketMap(m.allTickets)
	if len(byID) == 0 {
		byID = ticketMap(m.tickets)
	}
	definitions := []struct {
		name  string
		match func(Ticket) bool
	}{
		{name: "Ready", match: func(ticket Ticket) bool { return ticket.Status == "open" && isReady(ticket, byID) }},
		{name: "In Progress", match: func(ticket Ticket) bool { return ticket.Status == "in_progress" }},
		{name: "Blocked", match: func(ticket Ticket) bool { return ticket.Status == "open" && isBlocked(ticket, byID) }},
		{name: "Closed Recent", match: func(ticket Ticket) bool { return ticket.Status == "closed" }},
	}

	sections := make([]ticketSection, 0, len(definitions))
	for _, definition := range definitions {
		section := ticketSection{name: definition.name}
		for _, ticket := range m.tickets {
			if definition.match(ticket) {
				section.tickets = append(section.tickets, ticket)
			}
		}
		if len(section.tickets) > 0 {
			sections = append(sections, section)
		}
	}
	return sections
}

func ticketMap(tickets []Ticket) map[string]Ticket {
	byID := make(map[string]Ticket, len(tickets))
	for _, ticket := range tickets {
		byID[ticket.ID] = ticket
	}
	return byID
}

func (m model) orderedTickets() []Ticket {
	sections := m.visibleSections()
	if len(sections) == 0 {
		return m.tickets
	}
	ordered := make([]Ticket, 0, len(m.tickets))
	for _, section := range sections {
		ordered = append(ordered, section.tickets...)
	}
	return ordered
}

func (m model) currentSectionName() string {
	id := selectedID(m)
	for _, section := range m.visibleSections() {
		for _, ticket := range section.tickets {
			if ticket.ID == id {
				return section.name
			}
		}
	}
	return "-"
}

func (m model) nextSectionFirstIndex() int {
	id := selectedID(m)
	sections := m.visibleSections()
	if len(sections) == 0 {
		return 0
	}
	for sectionIndex, section := range sections {
		for _, ticket := range section.tickets {
			if ticket.ID == id {
				nextSectionIndex := (sectionIndex + 1) % len(sections)
				index := 0
				for i := 0; i < nextSectionIndex; i++ {
					index += len(sections[i].tickets)
				}
				return index
			}
		}
	}
	return 0
}

func (m model) renderTicketRow(ticket Ticket, width int) string {
	marker := " "
	if selectedID(m) == ticket.ID {
		marker = ">"
	}
	line := fmt.Sprintf("%s %-8s P%d %-11s %s", marker, ticket.ID, ticket.Priority, statusLabel(ticket.Status), ticket.Title)
	line = truncateText(line, width)
	if selectedID(m) == ticket.ID {
		return selectedStyle.Render(line)
	}
	if ticket.Status == "closed" {
		return mutedStyle.Render(line)
	}
	return line
}

func statusLabel(status string) string {
	return strings.ReplaceAll(status, "_", " ")
}

func (m model) updatePrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.prompt = promptNone
		m.input = newTextInput("")
		m.status = "Cancelled"
		m.err = ""
		return m, nil
	case "enter":
		input := strings.TrimSpace(m.input.Value())
		prompt := m.prompt
		m.prompt = promptNone
		m.input = newTextInput("")
		if input == "" {
			m.status = "Cancelled"
			return m, nil
		}
		if prompt == promptCreate {
			return m, m.createCmd(input)
		}
		return m, m.queryCmd(input)
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m model) renderPromptModal() string {
	if m.prompt == promptCreate {
		return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Create Ticket"),
			"Title",
			m.input.View(),
			mutedStyle.Render("enter create  esc cancel"),
		))
	}
	if m.prompt == promptQuery {
		return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Query Tickets"),
			"jq filter",
			m.input.View(),
			mutedStyle.Render("enter run query  esc cancel"),
		))
	}
	return ""
}

func newTextInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Focus()
	input.CharLimit = 200
	input.SetWidth(46)
	return input
}

func newPaletteInput() textinput.Model {
	input := newTextInput("Search commands")
	input.SetWidth(46)
	return input
}

func keyMsgFor(value string) tea.KeyPressMsg {
	switch value {
	case "tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	}
	return tea.KeyPressMsg(tea.Key{Code: []rune(value)[0], Text: value})
}

func (m *model) resizeDetail() {
	layout := layoutFor(m.width, m.height)
	m.detail.SetWidth(layout.detailWidth)
	m.detail.SetHeight(layout.detailHeight)
}

func (m model) loadCmd() tea.Cmd {
	return func() tea.Msg {
		tickets, err := LoadTickets(m.config.TicketsDir)
		if err != nil {
			return errorMsg{err: err}
		}
		detail := "No tickets. Run 'tk create' to add one."
		filtered := FilterTickets(tickets, m.mode)
		loadedModel := m
		loadedModel.allTickets = tickets
		loadedModel.tickets = filtered
		ordered := loadedModel.orderedTickets()
		if len(ordered) > 0 {
			selected := loadedModel.selected
			if selected >= len(ordered) {
				selected = len(ordered) - 1
			}
			if selected < 0 {
				selected = 0
			}
			detail = loadedModel.detailFor(ordered[selected].ID)
		}
		return loadedMsg{tickets: tickets, detail: detail}
	}
}

func refreshTickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (m model) loadDetailCmd() tea.Cmd {
	return func() tea.Msg {
		if len(m.tickets) == 0 || m.selected >= len(m.tickets) {
			return loadedMsg{tickets: m.allTickets, detail: "No tickets. Run 'tk create' to add one."}
		}
		return loadedMsg{tickets: m.allTickets, detail: m.detailFor(m.tickets[m.selected].ID)}
	}
}

func (m model) detailFor(id string) string {
	if m.config.TKScript == "" {
		return "TK_SCRIPT is not set; detail view requires launching through 'tk tui'."
	}
	output, err := m.runner(m.config.TKScript, "super", "show", id)
	if err != nil {
		return output + "\n" + err.Error()
	}
	layout := layoutFor(m.width, m.height)
	return RenderTicketDetail(output, m.config.TicketsDir, max(1, layout.detailWidth))
}

func (m model) actionCmd(action string) tea.Cmd {
	return func() tea.Msg {
		if len(m.tickets) == 0 || m.selected >= len(m.tickets) {
			return statusMsg{text: "No selected ticket"}
		}
		if m.config.TKScript == "" {
			return errorMsg{err: errors.New("TK_SCRIPT is not set; status actions require launching through 'tk tui'")}
		}
		id := m.tickets[m.selected].ID
		output, err := m.runner(m.config.TKScript, "super", action, id)
		if err != nil {
			return errorMsg{err: fmt.Errorf("%s %s failed: %s %w", action, id, output, err)}
		}
		return statusMsg{text: strings.TrimSpace(output)}
	}
}

func (m model) addDependenciesCmd(dependencyIDs []string) tea.Cmd {
	return func() tea.Msg {
		id := selectedID(m)
		if id == "" {
			return statusMsg{text: "No selected ticket"}
		}
		if len(dependencyIDs) == 0 {
			return statusMsg{text: "No dependencies selected"}
		}
		if m.config.TKScript == "" {
			return errorMsg{err: errors.New("TK_SCRIPT is not set; dependency actions require launching through 'tk tui'")}
		}
		var output strings.Builder
		for _, dependencyID := range dependencyIDs {
			text, err := m.runner(m.config.TKScript, "super", "dep", id, dependencyID)
			output.WriteString(text)
			if err != nil {
				return errorMsg{err: fmt.Errorf("dep %s %s failed: %s %w", id, dependencyID, text, err)}
			}
		}
		status := strings.TrimSpace(output.String())
		if status == "" {
			status = fmt.Sprintf("Added %d dependencies", len(dependencyIDs))
		}
		return statusMsg{text: status}
	}
}

func (m model) createCmd(title string) tea.Cmd {
	return func() tea.Msg {
		spec, err := commandSpec(m.config, "super", "create", title)
		if err != nil {
			return errorMsg{err: err}
		}
		output, err := m.runner(spec.name, spec.args...)
		if err != nil {
			return errorMsg{err: fmt.Errorf("create failed: %s %w", output, err)}
		}
		return statusMsg{text: "Created " + strings.TrimSpace(output)}
	}
}

func (m model) queryCmd(filter string) tea.Cmd {
	return func() tea.Msg {
		spec, err := commandSpec(m.config, "query", filter)
		if err != nil {
			return errorMsg{err: err}
		}
		output, err := m.runner(spec.name, spec.args...)
		if err != nil {
			return errorMsg{err: fmt.Errorf("query failed: %s %w", output, err)}
		}
		return queryMsg{filter: filter, output: output}
	}
}

type queryMsg struct {
	filter string
	output string
}

func (m model) editCmd() tea.Cmd {
	id := selectedID(m)
	if id == "" {
		return func() tea.Msg { return statusMsg{text: "No selected ticket"} }
	}
	cmd, err := editProcess(m.config, id)
	if err != nil {
		return func() tea.Msg {
			return errorMsg{err: err}
		}
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errorMsg{err: err}
		}
		return statusMsg{text: "Edited ticket"}
	})
}

func editProcess(config Config, id string) (*exec.Cmd, error) {
	if config.TKScript == "" {
		return nil, errors.New("TK_SCRIPT is not set; editing requires launching through 'tk tui'")
	}
	cmd := exec.Command(config.TKScript, "edit", id)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func selectedID(m model) string {
	if len(m.tickets) == 0 || m.selected >= len(m.tickets) {
		return ""
	}
	return m.tickets[m.selected].ID
}

func defaultRunner(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String() + stderr.String(), err
}

func commandSpec(config Config, args ...string) (commandSpecValue, error) {
	if config.TKScript == "" {
		return commandSpecValue{}, errors.New("TK_SCRIPT is not set; command actions require launching through 'tk tui'")
	}
	return commandSpecValue{name: config.TKScript, args: args}, nil
}

func EnvMap() map[string]string {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width])
}
