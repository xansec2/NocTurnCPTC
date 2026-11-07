package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenList screen = iota
	screenConfig
)

var (
	accent = lipgloss.Color("37")  // teal
	dim    = lipgloss.Color("244") // grey
	good   = lipgloss.Color("42")  // green
	warn   = lipgloss.Color("214") // amber
	bg     = lipgloss.Color("236")
	purple = lipgloss.Color("30") // panel borders (teal)

	mainBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(purple)
	outTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(purple)

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Padding(0, 1)
	flagSelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(bg)
	flagStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	descStyle    = lipgloss.NewStyle().Foreground(dim).Italic(true)
	onStyle      = lipgloss.NewStyle().Foreground(good).Bold(true)
	offStyle     = lipgloss.NewStyle().Foreground(dim)
	reqStyle     = lipgloss.NewStyle().Foreground(warn)
	cmdBoxStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1)
	cmdTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	helpStyle    = lipgloss.NewStyle().Foreground(dim)
	statusStyle  = lipgloss.NewStyle().Foreground(good).Bold(true)
	modeSelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(accent)
)

// jkTimeout is how soon k must follow j for the pair to leave insert mode, like
// vim's timeoutlen for an `inoremap jk <esc>` mapping. Slower, and both letters
// are typed literally.
const jkTimeout = 500 * time.Millisecond

type model struct {
	screen screen

	// Available tool catalogs ("modes": Web, AD, Linux, …) and the one in view.
	// Switching catalogs rebuilds the tool list in place.
	modes   []catalogMode
	modeIdx int

	list list.Model

	tool   Tool // working copy being configured
	cursor int  // index into tool.Flags
	input  textinput.Model
	status string // transient message (e.g. "copied!")

	// Vim-style modal state for the configurator.
	insert  bool      // insert mode: keys edit the selected value
	pending string    // first key of a two-key normal command (gg, dd, cc)
	lastJ   time.Time // when j was last typed in an insert context, for jk

	width, height int
	// panel geometry, recomputed on every resize.
	panelW, panelH int
	innerW         int

	// finalCmd is printed to stdout when the program quits.
	finalCmd string
}

func newModel() model {
	modes := catalogModes()
	// Start on the Web catalog if present, else the first one.
	start := 0
	for i, md := range modes {
		if md.Key == "web" {
			start = i
			break
		}
	}

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(accent).BorderForeground(accent)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(lipgloss.Color("250")).BorderForeground(accent)

	l := list.New(nil, d, 0, 0)
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.KeyMap.NextPage.SetKeys(append(l.KeyMap.NextPage.Keys(), "ctrl+d")...)
	l.KeyMap.PrevPage.SetKeys(append(l.KeyMap.PrevPage.Keys(), "ctrl+u")...)
	// A static cursor means filter updates never wait on a blink timer, which
	// acceptFilter relies on when it runs them synchronously.
	l.FilterInput.Cursor.SetMode(cursor.CursorStatic)
	// Advertise catalog switching in the list's help line.
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("L"), key.WithHelp("H/L", "switch catalog")),
		}
	}

	ti := textinput.New()
	ti.Prompt = "› "
	ti.CharLimit = 512

	m := model{
		screen: screenList,
		modes:  modes,
		list:   l,
		input:  ti,
	}
	m.loadMode(start)
	return m
}

// loadMode switches the active catalog (wrapping around) and rebuilds the tool
// list from it.
func (m *model) loadMode(idx int) {
	n := len(m.modes)
	if n == 0 {
		return
	}
	m.modeIdx = ((idx % n) + n) % n
	md := m.modes[m.modeIdx]
	tools := md.tools()
	items := make([]list.Item, len(tools))
	for i, t := range tools {
		items[i] = t
	}
	m.list.SetItems(items)
	m.list.Title = md.Label + " — pick a tool"
	m.list.ResetFilter()
	m.list.Select(0)
	m.screen = screenList
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// Single full-width panel: the tool catalog / configurator.
		m.panelW = msg.Width
		m.panelH = msg.Height

		m.innerW = maxInt(10, m.panelW-2) // minus the panel border
		toolInnerH := maxInt(3, m.panelH-2)
		// Leave room above the list for the catalog switcher.
		m.list.SetSize(m.innerW, maxInt(2, toolInnerH-2))
		m.input.Width = maxInt(8, m.innerW-28)
		return m, nil
	case tea.KeyMsg:
		if m.screen == screenList {
			return m.updateList(msg)
		}
		return m.updateConfig(msg)
	}
	// Non-key messages only matter to the list on the list screen.
	if m.screen == screenList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let the list own typing while its filter is active.
	if m.list.FilterState() == list.Filtering {
		if m.jkPressed(msg) {
			return m.acceptFilter()
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "L", "tab", "shift+right":
		// Next catalog (Setup → Footprinting → Web → AD → …).
		m.loadMode(m.modeIdx + 1)
		return m, nil
	case "H", "shift+tab", "shift+left":
		// Previous catalog.
		m.loadMode(m.modeIdx - 1)
		return m, nil
	case "enter":
		if sel, ok := m.list.SelectedItem().(Tool); ok {
			m.tool = sel.clone()
			m.cursor = 0
			m.status = ""
			m.insert = false
			m.pending = ""
			m.syncInput()
			m.screen = screenConfig
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// jkPressed tracks typed keys in an insert context and reports whether this one
// is the k that completes jk. The j has already been typed by then, so the caller
// must remove it. Pasted text never matches: its String() is bracketed.
func (m *model) jkPressed(msg tea.KeyMsg) bool {
	s := msg.String()
	done := s == "k" && !m.lastJ.IsZero() && time.Since(m.lastJ) <= jkTimeout
	m.lastJ = time.Time{}
	if s == "j" {
		m.lastJ = time.Now()
	}
	return done
}

// acceptFilter leaves the list's filter input, keeping the query minus the j of
// jk. The shortened query is matched before accepting because the list drops a
// filter that has no matches, and the query with the j often has none.
func (m model) acceptFilter() (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m.list = settleList(m.list, cmd)
	m.list, cmd = m.list.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return m, cmd
}

// settleList runs cmd now and feeds the resulting messages back into the list.
func settleList(l list.Model, cmd tea.Cmd) list.Model {
	if cmd == nil {
		return l
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			l = settleList(l, c)
		}
		return l
	}
	l, _ = l.Update(msg)
	return l
}

func (m model) updateConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Some tools (e.g. the bare GTFOBins shell spawns) carry no flags: there is
	// nothing to configure, just a command to copy.
	if len(m.tool.Flags) == 0 {
		return m.updateConfigNoFlags(msg)
	}
	if m.insert {
		return m.updateInsert(msg)
	}
	return m.updateNormal(msg)
}

// updateInsert handles insert mode: keys edit the selected value until esc or jk.
func (m model) updateInsert(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := &m.tool.Flags[m.cursor]
	if m.jkPressed(msg) {
		m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		cur.Value = m.input.Value()
		m.leaveInsert()
		return m, nil
	}

	s := msg.String()
	if cmd, ok := m.commandKey(s); ok {
		return m, cmd
	}
	switch s {
	case "esc":
		m.leaveInsert()
		return m, nil
	case "up", "shift+tab":
		m.leaveInsert()
		m.move(-1)
		return m, nil
	case "down", "tab":
		m.leaveInsert()
		m.move(1)
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cur.Value = m.input.Value()
	m.status = ""
	return m, cmd
}

// updateNormal handles normal mode: keys navigate and run commands, never type.
func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := &m.tool.Flags[m.cursor]
	isText := cur.Kind == KindString || cur.Kind == KindInt

	s := msg.String()
	if cmd, ok := m.commandKey(s); ok {
		m.pending = ""
		return m, cmd
	}

	// Two-key commands: as in vim, any other second key cancels.
	if m.pending != "" {
		first := m.pending
		m.pending = ""
		if s != first {
			return m, nil
		}
		s = first + s
	} else if s == "g" || s == "d" || s == "c" {
		m.pending = s
		return m, nil
	}

	switch s {
	case "esc", "q":
		m.screen = screenList
		m.status = ""
	case "k", "up", "shift+tab":
		m.move(-1)
	case "j", "down", "tab":
		m.move(1)
	case "gg":
		m.moveTo(0)
	case "G":
		m.moveTo(len(m.tool.Flags) - 1)
	case "y":
		m.yank()
	case " ":
		if cur.Kind == KindBool {
			cur.Enabled = !cur.Enabled
			m.status = ""
		}
	case "enter":
		switch {
		case cur.Kind == KindBool:
			cur.Enabled = !cur.Enabled
			m.status = ""
		case cur.Kind == KindChoice:
			m.cycleChoice(cur, 1)
		case isText:
			m.enterInsert(m.input.Position())
		}
	case "h", "left":
		if cur.Kind == KindChoice {
			m.cycleChoice(cur, -1)
		} else if isText {
			m.setInputPos(m.input.Position() - 1)
		}
	case "l", "right":
		if cur.Kind == KindChoice {
			m.cycleChoice(cur, 1)
		} else if isText {
			m.setInputPos(m.input.Position() + 1)
		}
	}
	if !isText {
		return m, nil
	}

	// Commands that only make sense on an editable value.
	value := []rune(m.input.Value())
	pos := m.input.Position()
	switch s {
	case "0":
		m.setInputPos(0)
	case "$":
		m.setInputPos(len(value))
	case "w":
		m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true})
		m.setInputPos(m.input.Position())
	case "b":
		m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
	case "i":
		m.enterInsert(pos)
	case "a":
		m.enterInsert(pos + 1)
	case "I":
		m.enterInsert(0)
	case "A":
		m.enterInsert(len(value))
	case "x":
		if pos < len(value) {
			m.setValue(cur, string(append(value[:pos:pos], value[pos+1:]...)), pos)
		}
	case "D":
		m.setValue(cur, string(value[:pos]), pos)
	case "C":
		m.setValue(cur, string(value[:pos]), pos)
		m.enterInsert(pos)
	case "dd":
		m.setValue(cur, "", 0)
	case "cc", "S":
		m.setValue(cur, "", 0)
		m.enterInsert(0)
	}
	return m, nil
}

// updateConfigNoFlags handles the config screen for a flag-less tool: there is
// nothing to navigate or edit, so only the command-level actions apply.
func (m model) updateConfigNoFlags(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if cmd, ok := m.commandKey(s); ok {
		return m, cmd
	}
	switch s {
	case "esc", "q":
		m.screen = screenList
		m.status = ""
	case "y":
		m.yank()
	}
	return m, nil
}

// commandKey runs the command-level actions available in every config mode.
func (m *model) commandKey(s string) (tea.Cmd, bool) {
	switch s {
	case "ctrl+c":
		return tea.Quit, true
	case "ctrl+y":
		m.yank()
		return nil, true
	case "ctrl+o":
		m.finalCmd = m.tool.Command()
		return tea.Quit, true
	}
	return nil, false
}

// yank copies the assembled command to the clipboard.
func (m *model) yank() {
	cmd := m.tool.Command()
	m.finalCmd = cmd
	if err := clipboard.WriteAll(cmd); err != nil {
		m.status = "clipboard unavailable — command printed on exit (ctrl+o)"
	} else {
		m.status = "copied to clipboard ✓"
	}
}

func (m *model) enterInsert(pos int) {
	m.insert = true
	m.lastJ = time.Time{}
	m.input.SetCursor(pos)
	m.status = ""
}

// leaveInsert returns to normal mode, stepping the cursor back one like vim.
func (m *model) leaveInsert() {
	m.insert = false
	m.lastJ = time.Time{}
	m.setInputPos(m.input.Position() - 1)
}

// setInputPos moves the cursor for normal mode, which rests on a character rather
// than past the last one.
func (m *model) setInputPos(pos int) {
	m.input.SetCursor(minInt(pos, len([]rune(m.input.Value()))-1))
}

// setValue replaces the selected flag's value during a normal-mode edit.
func (m *model) setValue(f *Flag, value string, pos int) {
	m.input.SetValue(value)
	m.setInputPos(pos)
	f.Value = value
	m.status = ""
}

func (m *model) move(delta int) {
	n := len(m.tool.Flags)
	if n == 0 {
		return
	}
	m.moveTo((m.cursor + delta + n) % n)
}

func (m *model) moveTo(i int) {
	m.cursor = i
	m.syncInput()
	m.status = ""
}

// syncInput points the textinput at the currently selected flag (if editable).
func (m *model) syncInput() {
	if len(m.tool.Flags) == 0 {
		m.input.Blur()
		return
	}
	cur := m.tool.Flags[m.cursor]
	if cur.Kind == KindString || cur.Kind == KindInt {
		m.input.SetValue(cur.Value)
		m.input.Placeholder = cur.Placeholder
		m.setInputPos(len([]rune(cur.Value)))
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

func (m *model) cycleChoice(f *Flag, delta int) {
	if len(f.Choices) == 0 {
		return
	}
	idx := 0
	for i, c := range f.Choices {
		if c == f.Value {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(f.Choices)) % len(f.Choices)
	f.Value = f.Choices[idx]
	m.status = ""
}

func (m model) View() string {
	if m.width == 0 {
		return "initializing…"
	}
	return m.toolPanel()
}

// modeBar is the catalog switcher shown above the tool list.
func (m model) modeBar() string {
	md := m.modes[m.modeIdx]
	bar := outTitleStyle.Render("catalog ") + modeSelStyle.Render(" "+md.Label+" ")
	// Catalogs only switch from the list screen.
	if m.screen == screenList {
		bar += "  " + helpStyle.Render(fmt.Sprintf("(%d/%d · H/L)", m.modeIdx+1, len(m.modes)))
	}
	return bar
}

// toolPanel is the full-width panel: the catalog switcher over the tool list or
// the tool configurator.
func (m model) toolPanel() string {
	content := m.list.View()
	if m.screen == screenConfig {
		content = m.configView()
	}
	body := m.modeBar() + "\n" + content
	return mainBorderStyle.
		Width(m.panelW - 2).Height(m.panelH - 2).
		Render(body)
}

func (m model) configView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf(" %s — %s ", m.tool.Name, m.tool.Desc)))
	b.WriteString("\n\n")

	for i, f := range m.tool.Flags {
		selected := i == m.cursor
		cursor := "  "
		if selected {
			cursor = lipgloss.NewStyle().Foreground(accent).Render("▸ ")
		}

		label := f.label()
		if f.Required {
			label += reqStyle.Render(" *")
		}

		var val string
		switch f.Kind {
		case KindBool:
			if f.Enabled {
				val = onStyle.Render("[x] on")
			} else {
				val = offStyle.Render("[ ] off")
			}
		case KindChoice:
			val = fmt.Sprintf("‹ %s ›", f.Value)
		default:
			if selected {
				val = m.input.View()
			} else {
				val = flagStyle.Render(f.summary())
			}
		}

		line := fmt.Sprintf("%s%-26s %s", cursor, label, val)
		if selected {
			line = flagSelStyle.Render(line)
		}
		b.WriteString(line + "\n")

		if selected && f.Desc != "" {
			b.WriteString("    " + descStyle.Render(f.Desc) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.commandBox())
	b.WriteString("\n")

	if m.status != "" {
		b.WriteString(statusStyle.Render(m.status) + "\n")
	}
	b.WriteString(helpStyle.Render(m.configHelp()))

	return b.String()
}

func (m model) configHelp() string {
	switch {
	case len(m.tool.Flags) == 0:
		return "y copy · ctrl+o print & quit · q/esc back"
	case m.insert:
		return "-- INSERT -- jk/esc normal · ctrl+y copy · ctrl+o print & quit"
	}
	return "-- NORMAL -- j/k move · h/l cursor/choice · space toggle · i/a insert · x/D/dd delete · y copy · ctrl+o print & quit · q/esc back"
}

func (m model) commandBox() string {
	cmd := m.tool.Command()
	box := cmdBoxStyle.Width(maxInt(20, m.panelW-6))
	content := cmdTextStyle.Render(cmd)
	if missing := m.tool.MissingRequired(); len(missing) > 0 {
		content += "\n" + reqStyle.Render("missing required: "+strings.Join(missing, ", "))
	}
	return box.Render(content)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
