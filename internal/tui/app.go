package tui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	config       Config
	runner       commandRunner
	width        int
	height       int
	focus        focusArea
	allTickets   []Ticket
	tickets      []Ticket
	selected     int
	mode         Mode
	detail       viewport.Model
	detailHeader string
	status       string
	err          string
	prompt       promptKind
	input        textinput.Model
	queryShown   bool
	helpShown    bool
	palette      paletteState
	depPicker    dependencyPickerState
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
	detail  detailParts
}

type detailLoadedMsg struct {
	detail detailParts
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
	ansiPattern   = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

const (
	narrowLayoutWidth = 80
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
		if selectedID(m) != "" && m.prompt == promptNone && !m.queryShown && !m.helpShown && !m.palette.shown && !m.depPicker.shown {
			return m, m.loadDetailCmd()
		}
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
		m.applyDetail(msg.detail)
		m.status = fmt.Sprintf("%d tickets", len(m.tickets))
		m.err = ""
		if m.width > 0 && len(m.tickets) > 0 {
			return m, m.loadDetailCmd()
		}
		return m, nil
	case detailLoadedMsg:
		m.applyDetail(msg.detail)
		return m, nil
	case statusMsg:
		m.status = msg.text
		m.err = ""
		return m, m.loadCmd()
	case errorMsg:
		m.err = msg.err.Error()
		return m, nil
	case queryMsg:
		m.detailHeader = ""
		m.resizeDetail()
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
				m.resizeDetail()
				return m, m.loadDetailCmd()
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
				m.resizeDetail()
				return m, m.loadDetailCmd()
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

	header := titleStyle.Render(m.headerText())
	layout := layoutFor(m.width, m.height)

	listTitle := titleStyle.Render("Queue")
	listContent := lipgloss.NewStyle().Width(layout.listWidth).Height(layout.listHeight).Render(m.renderList(layout.listWidth, layout.listHeight))
	list := lipgloss.JoinVertical(lipgloss.Left, listTitle, listContent)
	detailContent := m.detail.View()
	detailSections := []string{titleStyle.Render("Ticket")}
	if m.detailHeader != "" {
		detailSections = append(detailSections, m.detailHeader)
	}
	if m.detailHeader != "" && detailContent != "" {
		detailSections = append(detailSections, "")
	}
	detailSections = append(detailSections, detailContent)
	detailInner := lipgloss.JoinVertical(lipgloss.Left, detailSections...)
	detail := lipgloss.NewStyle().Width(layout.detailWidth).Height(layout.detailHeight).Render(detailInner)
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

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footerStyle.Render(footerText))
	if overlay := m.activeOverlay(); overlay != "" {
		content = overlayOnScreen(content, overlay, m.width, layout.bodyHeight, 2)
	}
	if m.focus == focusPreview {
		content = overlayOnScreen(content, m.renderPreviewModal(), m.width, layout.bodyHeight, 2)
	}

	view := tea.NewView(content)
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
	footerText := m.mainFooterFor()
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

func (m model) mainFooterFor() string {
	parts := []string{"j/k move", "enter inspect", "tab section", "n new", "d deps", "/ filter", "? help"}
	if m.width > 110 {
		parts = []string{"j/k move", "enter inspect", "tab section", "n new", "d deps", "/ filter", "ctrl+p cmds", "? help"}
	}
	return strings.Join(parts, "   ")
}

func (m model) headerText() string {
	parts := []string{"tk · tickets"}
	sections := m.visibleSections()
	for _, name := range []string{"Ready", "In Progress", "Blocked", "Closed Recent"} {
		count := sectionCount(sections, name)
		if count == 0 {
			continue
		}
		switch name {
		case "Ready":
			parts = append(parts, fmt.Sprintf("Ready %d", count))
		case "In Progress":
			parts = append(parts, fmt.Sprintf("Active %d", count))
		case "Blocked":
			parts = append(parts, fmt.Sprintf("Blocked %d", count))
		case "Closed Recent":
			parts = append(parts, fmt.Sprintf("Closed %d", count))
		}
	}
	return strings.Join(parts, "   ")
}

func compactFooterStatus(status string) string {
	if status == "Focused ticket list" {
		return "list"
	}
	return status
}

func (m model) activeOverlay() string {
	switch {
	case m.prompt != promptNone:
		return m.renderPromptModal()
	case m.depPicker.shown:
		return m.renderDependencyPickerModal()
	case m.palette.shown:
		return m.renderPaletteModal()
	case m.helpShown:
		return renderHelpModal()
	default:
		return ""
	}
}

func (m model) renderPreviewModal() string {
	title := "Preview"
	if id := selectedID(m); id != "" {
		title = "Preview: " + id
	}
	detailSections := []string{title}
	if m.detailHeader != "" {
		detailSections = append(detailSections, m.detailHeader, "")
	}
	if body := m.detail.View(); body != "" {
		detailSections = append(detailSections, body)
	}
	detailSections = append(detailSections, "", "esc close   j/k scroll   e edit")

	width, height := m.previewModalDimensions()
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(max(1, width-6)).
		Height(max(1, height-4)).
		Render(lipgloss.JoinVertical(lipgloss.Left, detailSections...))
}

func (m model) previewModalDimensions() (int, int) {
	width := max(40, m.width-20)
	height := max(10, m.height-10)
	if width > 90 {
		width = 90
	}
	return width, height
}

func overlayOnScreen(screen string, overlay string, width int, height int, topRow int) string {
	screenLines := strings.Split(screen, "\n")
	overlayLines := strings.Split(strings.TrimRight(overlay, "\n"), "\n")
	if len(screenLines) == 0 || len(overlayLines) == 0 {
		return screen
	}
	overlayWidth := 0
	for _, line := range overlayLines {
		if w := len([]rune(stripANSIText(line))); w > overlayWidth {
			overlayWidth = w
		}
	}
	startRow := max(0, topRow-1+max(0, (height-len(overlayLines))/2))
	startCol := max(0, (width-overlayWidth)/2)
	for i, line := range overlayLines {
		idx := startRow + i
		if idx >= len(screenLines) {
			break
		}
		prefix := ansi.Cut(screenLines[idx], 0, startCol)
		suffixWidth := max(0, width-startCol-overlayWidth)
		suffix := strings.Repeat(" ", suffixWidth)
		padding := overlayWidth - len([]rune(stripANSIText(line)))
		if padding > 0 {
			line += strings.Repeat(" ", padding)
		}
		screenLines[idx] = prefix + line + suffix
	}
	return strings.Join(screenLines, "\n")
}

func previewOverlaySize(lines []string) (int, int) {
	width := 0
	for _, line := range lines {
		if w := len([]rune(line)); w > width {
			width = w
		}
	}
	return width, len(lines)
}

func stripANSIText(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
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

	lines, selectedLine := m.renderListLines(width)
	if len(lines) <= height {
		return strings.Join(lines, "\n")
	}

	start := 0
	if selectedLine >= 0 && selectedLine >= height {
		start = selectedLine - height + 1
	}
	if start > len(lines)-height {
		start = len(lines) - height
	}
	return strings.Join(lines[start:start+height], "\n")
}

func (m model) renderListLines(width int) ([]string, int) {
	sections := m.visibleSections()
	cols := m.listColumns(sections)
	lines := make([]string, 0, len(m.tickets)+len(sections))
	selectedLine := -1
	for _, section := range sections {
		lines = append(lines, titleStyle.Render(truncateText(fmt.Sprintf("─ %s (%d)", section.name, len(section.tickets)), width)))
		for _, ticket := range section.tickets {
			if selectedID(m) == ticket.ID {
				selectedLine = len(lines)
			}
			lines = append(lines, m.renderTicketRow(ticket, width, cols))
		}
	}
	return lines, selectedLine
}

type ticketSection struct {
	name    string
	tickets []Ticket
}

type listColumns struct {
	idWidth    int
	stateWidth int
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

func sectionCount(sections []ticketSection, name string) int {
	for _, section := range sections {
		if section.name == name {
			return len(section.tickets)
		}
	}
	return 0
}

func unresolvedDepCount(ticket Ticket, byID map[string]Ticket) int {
	count := 0
	for _, depID := range ticket.Deps {
		dep, ok := byID[depID]
		if !ok || dep.Status != "closed" {
			count++
		}
	}
	return count
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

func (m model) renderTicketRow(ticket Ticket, width int, cols listColumns) string {
	marker := " "
	if selectedID(m) == ticket.ID {
		marker = ">"
	}
	state := m.ticketStateLabel(ticket)
	idCol := padListCell(ticket.ID, cols.idWidth)
	stateCol := padListCell(state, cols.stateWidth)
	prefix := fmt.Sprintf("%s P%d  ", marker, ticket.Priority)
	suffix := "  " + idCol + "  " + stateCol
	titleWidth := max(1, width-len(prefix)-len(suffix))
	title := truncateText(ticket.Title, titleWidth)
	line := prefix + padListCell(title, titleWidth) + suffix
	line = truncateText(line, width)
	if selectedID(m) == ticket.ID {
		return selectedStyle.Render(line)
	}
	if ticket.Status == "closed" {
		return mutedStyle.Render(line)
	}
	return line
}

func (m model) listColumns(sections []ticketSection) listColumns {
	cols := listColumns{idWidth: 8, stateWidth: 9}
	for _, section := range sections {
		for _, ticket := range section.tickets {
			if w := len(ticket.ID); w > cols.idWidth {
				cols.idWidth = w
			}
			if w := len(m.ticketStateLabel(ticket)); w > cols.stateWidth {
				cols.stateWidth = w
			}
		}
	}
	return cols
}

func padListCell(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func (m model) ticketStateLabel(ticket Ticket) string {
	if ticket.Status == "closed" {
		return "closed"
	}
	if ticket.Status == "in_progress" {
		return "active"
	}
	byID := ticketMap(m.allTickets)
	if unresolved := unresolvedDepCount(ticket, byID); unresolved > 0 {
		return fmt.Sprintf("blocked·%d", unresolved)
	}
	return "ready"
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

func (m model) loadDetailCmd() tea.Cmd {
	return func() tea.Msg {
		if len(m.tickets) == 0 || m.selected >= len(m.tickets) {
			return detailLoadedMsg{detail: detailParts{Body: "No tickets. Run 'tk create' to add one."}}
		}
		return detailLoadedMsg{detail: m.detailFor(m.tickets[m.selected].ID)}
	}
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

func (m *model) applyDetail(detail detailParts) {
	m.detailHeader = detail.Header
	m.resizeDetail()
	m.detail.SetContent(detail.Body)
}

func (m *model) resizeDetail() {
	detailWidth, detailHeight := m.detailDimensions()
	m.detail.SetWidth(detailWidth)
	headerLines := 0
	if m.detailHeader != "" {
		headerLines = strings.Count(m.detailHeader, "\n") + 1
	}
	available := detailHeight - headerLines - 1
	m.detail.SetHeight(max(1, available))
}

func (m model) detailDimensions() (int, int) {
	layout := layoutFor(m.width, m.height)
	if m.focus == focusPreview {
		width, height := m.previewModalDimensions()
		return width - 4, height - 4
	}
	return layout.detailWidth, layout.detailHeight
}

func (m model) loadCmd() tea.Cmd {
	return func() tea.Msg {
		tickets, err := LoadTickets(m.config.TicketsDir)
		if err != nil {
			return errorMsg{err: err}
		}
		detail := detailParts{Body: "No tickets. Run 'tk create' to add one."}
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

func (m model) detailFor(id string) detailParts {
	if m.config.TKScript == "" {
		return detailParts{Body: "TK_SCRIPT is not set; detail view requires launching through 'tk tui'."}
	}
	output, err := m.runner(m.config.TKScript, "super", "show", id)
	if err != nil {
		return detailParts{Body: output + "\n" + err.Error()}
	}
	detailWidth, _ := m.detailDimensions()
	return RenderTicketDetailParts(output, m.config.TicketsDir, max(1, detailWidth))
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
