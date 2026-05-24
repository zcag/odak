package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zcag/odak/internal/client"
	"github.com/zcag/odak/internal/model"
)

// ── palette ───────────────────────────────────────────────────────────────────
// One accent, three neutrals (text / dim / very-dim), status colors.
// No background fills — selection is purely typographic.

var (
	clrAccent  = lipgloss.AdaptiveColor{Light: "#4f46e5", Dark: "#818cf8"}
	clrText    = lipgloss.AdaptiveColor{Light: "#1e293b", Dark: "#e2e8f0"}
	clrDim     = lipgloss.AdaptiveColor{Light: "#64748b", Dark: "#64748b"}
	clrFaint   = lipgloss.AdaptiveColor{Light: "#94a3b8", Dark: "#334155"}
	clrDone    = lipgloss.AdaptiveColor{Light: "#94a3b8", Dark: "#475569"}
	clrUrgent  = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"}
	clrBorder  = lipgloss.AdaptiveColor{Light: "#e2e8f0", Dark: "#1e293b"}
	clrGreen   = lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#4ade80"}
	clrRed     = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"}
)

// ── messages ──────────────────────────────────────────────────────────────────

type (
	allDataMsg struct {
		sections []client.SectionInfo
		items    []*model.Item
	}
	doneMsg struct{}
	errMsg  struct{ err error }
)

// ── rows ──────────────────────────────────────────────────────────────────────

type rowKind int

const (
	rowSpacer  rowKind = iota // blank line between sections
	rowSection                // section header
	rowItem                   // todo item
)

type row struct {
	kind    rowKind
	secName string
	count   int
	item    *model.Item
}

// ── state ─────────────────────────────────────────────────────────────────────

type modeID int

const (
	modeBrowse modeID = iota
	modeAdd
	modeMove
	modeConfirmDelete
	modeEdit
)

type Model struct {
	cl            *client.Client
	sections      []client.SectionInfo
	allItems      []*model.Item
	cursor        int
	offset        int
	filterSection string
	mode          modeID
	input         textinput.Model
	addSection    string
	editID        string
	showDone      bool
	w, h          int
	loading       bool
	err           string
}

func New(cl *client.Client) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(clrDim)
	ti.TextStyle = lipgloss.NewStyle().Foreground(clrText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(clrAccent)
	return Model{cl: cl, input: ti, loading: true, showDone: true}
}

func Run(cl *client.Client) error {
	_, err := tea.NewProgram(New(cl), tea.WithAltScreen()).Run()
	return err
}

func (m Model) Init() tea.Cmd { return m.fetchAll() }

// ── update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case allDataMsg:
		m.sections = msg.sections
		m.allItems = msg.items
		m.loading = false
		m = m.clampScroll()
		return m, nil
	case doneMsg:
		m.err = ""
		return m, m.fetchAll()
	case errMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.mode == modeAdd || m.mode == modeEdit {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) clampScroll() Model {
	rows := m.visibleRows()
	if m.cursor >= len(rows) {
		m.cursor = imax(0, len(rows)-1)
	}
	bh := m.bodyH()
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
	if bh > 0 && m.offset+bh <= m.cursor {
		m.offset = m.cursor - bh + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	return m
}

// ── key handling ──────────────────────────────────────────────────────────────

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeAdd:
		return m.handleAddKey(k)
	case modeEdit:
		return m.handleEditKey(k)
	case modeMove:
		return m.handleMoveKey(k)
	case modeConfirmDelete:
		return m.handleDeleteKey(k)
	}
	return m.handleBrowseKey(k)
}

func (m Model) handleBrowseKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.visibleRows()
	bh := m.bodyH()

	// skip returns the next non-spacer index from start in direction dir (+1/-1)
	skip := func(start, dir int) int {
		i := start
		for i >= 0 && i < len(rows) && rows[i].kind == rowSpacer {
			i += dir
		}
		if i < 0 || i >= len(rows) {
			return start
		}
		return i
	}

	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		next := skip(m.cursor+1, 1)
		if next < len(rows) {
			m.cursor = next
			if m.offset+bh <= m.cursor {
				m.offset = m.cursor - bh + 1
			}
		}

	case "k", "up":
		prev := skip(m.cursor-1, -1)
		if prev >= 0 && prev != m.cursor {
			m.cursor = prev
			if m.offset > m.cursor {
				m.offset = m.cursor
			}
		}

	case "g":
		m.cursor = skip(0, 1)
		m.offset = 0

	case "G":
		m.cursor = skip(len(rows)-1, -1)
		if bh > 0 && m.offset+bh <= m.cursor {
			m.offset = m.cursor - bh + 1
		}

	case "enter", "f":
		if sec := m.rowSection(rows); sec != "" && sec != m.filterSection {
			m.filterSection = sec
			fr := m.visibleRows()
			// land on first item after section header
			m.cursor = skip(1, 1)
			if m.cursor >= len(fr) {
				m.cursor = 0
			}
			m.offset = 0
		}

	case "esc":
		if m.filterSection != "" {
			sec := m.filterSection
			m.filterSection = ""
			full := m.visibleRows()
			for i, r := range full {
				if r.kind == rowSection && r.secName == sec {
					m.cursor = i
					break
				}
			}
			m = m.clampScroll()
		}

	case " ", "x":
		if item := m.currentItem(rows); item != nil {
			return m, m.cmdToggle(item.ID)
		}

	case "a":
		sec := m.rowSection(rows)
		if sec == "" && len(m.sections) > 0 {
			sec = m.sections[0].Name
		}
		m.addSection = sec
		m.input.Placeholder = "new item…"
		m.input.SetValue("")
		m.input.Focus()
		m.mode = modeAdd
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(nil)
		return m, cmd

	case "e":
		if item := m.currentItem(rows); item != nil {
			m.editID = item.ID
			m.input.Placeholder = "edit text…"
			m.input.SetValue(item.Text)
			m.input.Focus()
			m.input.CursorEnd()
			m.mode = modeEdit
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(nil)
			return m, cmd
		}

	case "h":
		m.showDone = !m.showDone
		m = m.clampScroll()

	case "d":
		if m.currentItem(rows) != nil {
			m.mode = modeConfirmDelete
		}

	case "m":
		if m.currentItem(rows) != nil {
			m.mode = modeMove
		}

	case "r":
		m.loading = true
		return m, m.fetchAll()
	}
	return m, nil
}

func (m Model) handleAddKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEnter:
		text := strings.TrimSpace(m.input.Value())
		m.mode = modeBrowse
		m.input.Blur()
		if text != "" {
			return m, m.cmdAdd(text, m.addSection)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = modeBrowse
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(k)
	return m, cmd
}

func (m Model) handleEditKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEnter:
		text := strings.TrimSpace(m.input.Value())
		id := m.editID
		m.mode = modeBrowse
		m.editID = ""
		m.input.Blur()
		if text != "" {
			return m, m.cmdUpdate(id, text)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = modeBrowse
		m.editID = ""
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(k)
	return m, cmd
}

func (m Model) handleMoveKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyEsc || k.String() == "q" {
		m.mode = modeBrowse
		return m, nil
	}
	if k.String() >= "1" && k.String() <= "9" {
		idx := int(k.String()[0] - '1')
		if idx < len(m.sections) {
			if item := m.currentItem(m.visibleRows()); item != nil {
				m.mode = modeBrowse
				return m, m.cmdMove(item.ID, m.sections[idx].Name)
			}
		}
	}
	return m, nil
}

func (m Model) handleDeleteKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "y", "Y":
		m.mode = modeBrowse
		if item := m.currentItem(m.visibleRows()); item != nil {
			return m, m.cmdDelete(item.ID)
		}
	default:
		m.mode = modeBrowse
	}
	return m, nil
}

// ── row helpers ───────────────────────────────────────────────────────────────

func (m Model) visibleRows() []row {
	var rows []row
	first := true
	for _, sec := range m.sections {
		if m.filterSection != "" && sec.Name != m.filterSection {
			continue
		}
		if !first {
			rows = append(rows, row{kind: rowSpacer})
		}
		first = false
		rows = append(rows, row{kind: rowSection, secName: sec.Name, count: sec.Count})
		for _, item := range m.allItems {
			if item.Section == sec.Name && (m.showDone || !item.Done) {
				rows = append(rows, row{kind: rowItem, item: item})
			}
		}
	}
	return rows
}

func (m Model) currentItem(rows []row) *model.Item {
	if m.cursor < len(rows) && rows[m.cursor].kind == rowItem {
		return rows[m.cursor].item
	}
	return nil
}

func (m Model) rowSection(rows []row) string {
	if m.cursor >= len(rows) {
		return ""
	}
	r := rows[m.cursor]
	switch r.kind {
	case rowSection:
		return r.secName
	case rowItem:
		if r.item != nil {
			return r.item.Section
		}
	}
	return ""
}

// ── commands ──────────────────────────────────────────────────────────────────

func (m Model) fetchAll() tea.Cmd {
	return func() tea.Msg {
		secs, err := m.cl.Sections()
		if err != nil {
			return errMsg{err}
		}
		items, err := m.cl.List("", "", "")
		if err != nil {
			return errMsg{err}
		}
		return allDataMsg{sections: secs, items: items}
	}
}

func (m Model) cmdToggle(id string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.cl.ToggleDone(id); err != nil {
			return errMsg{err}
		}
		return doneMsg{}
	}
}

func (m Model) cmdAdd(text, sec string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.cl.Create(&model.Item{Text: text, Section: sec}); err != nil {
			return errMsg{err}
		}
		return doneMsg{}
	}
}

func (m Model) cmdDelete(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.cl.Delete(id); err != nil {
			return errMsg{err}
		}
		return doneMsg{}
	}
}

func (m Model) cmdUpdate(id, text string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.cl.Update(id, &model.Item{Text: text}); err != nil {
			return errMsg{err}
		}
		return doneMsg{}
	}
}

func (m Model) cmdMove(id, section string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.cl.Move(id, section); err != nil {
			return errMsg{err}
		}
		return doneMsg{}
	}
}

// ── view constants ────────────────────────────────────────────────────────────

const (
	maxW   = 110
	margin = 2 // left margin for all rows
)

func (m Model) ew() (w, lp int) {
	w = imin(m.w, maxW)
	lp = (m.w - w) / 2
	return
}

func (m Model) bodyH() int {
	if m.h < 5 {
		return 1
	}
	return m.h - 4 // header + top-div + bot-div + footer
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.w == 0 {
		return ""
	}
	w, lp := m.ew()
	bh := m.bodyH()
	rows := m.visibleRows()

	div := lipgloss.NewStyle().Foreground(clrBorder).Render(strings.Repeat("─", w))

	lines := []string{
		m.renderTitle(w),
		div,
		m.renderBody(rows, w, bh),
		div,
		m.renderFooter(w),
	}
	content := strings.Join(lines, "\n")

	if lp <= 0 {
		return content
	}
	pad := strings.Repeat(" ", lp)
	var sb strings.Builder
	for i, line := range strings.Split(content, "\n") {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(pad)
		sb.WriteString(line)
	}
	return sb.String()
}

func (m Model) renderTitle(w int) string {
	brand := lipgloss.NewStyle().Foreground(clrAccent).Bold(true).Render("odak")

	var left string
	if m.filterSection != "" {
		sep := lipgloss.NewStyle().Foreground(clrDim).Render("  /  ")
		sec := lipgloss.NewStyle().Foreground(clrText).Render(m.filterSection)
		left = strings.Repeat(" ", margin) + brand + sep + sec
	} else {
		left = strings.Repeat(" ", margin) + brand
	}

	open := 0
	for _, it := range m.allItems {
		if !it.Done {
			open++
		}
	}
	var summary string
	if m.filterSection != "" {
		for _, it := range m.allItems {
			if it.Section == m.filterSection && !it.Done {
				open++
			}
		}
		summary = fmt.Sprintf("%d open", open)
	} else {
		summary = fmt.Sprintf("%d open", open)
	}
	right := lipgloss.NewStyle().Foreground(clrDim).Render(summary) + strings.Repeat(" ", margin)

	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := w - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderBody(rows []row, w, bh int) string {
	if m.loading {
		lines := make([]string, bh)
		for i := range lines {
			lines[i] = ""
		}
		lines[0] = strings.Repeat(" ", margin) + lipgloss.NewStyle().Foreground(clrDim).Render("loading…")
		return strings.Join(lines, "\n")
	}

	if len(rows) == 0 {
		lines := make([]string, bh)
		for i := range lines {
			lines[i] = ""
		}
		lines[0] = strings.Repeat(" ", margin) + lipgloss.NewStyle().Foreground(clrDim).Render("nothing here")
		return strings.Join(lines, "\n")
	}

	end := imin(m.offset+bh, len(rows))
	visible := rows[m.offset:end]

	lines := make([]string, bh)
	for i, r := range visible {
		absIdx := m.offset + i
		sel := absIdx == m.cursor
		switch r.kind {
		case rowSpacer:
			lines[i] = ""
		case rowSection:
			lines[i] = m.renderSection(r, w, sel)
		case rowItem:
			lines[i] = m.renderItem(r.item, sel, w)
		}
	}
	// remaining lines already ""
	return strings.Join(lines, "\n")
}

func (m Model) renderSection(r row, w int, sel bool) string {
	// "  name                                                count  "
	pad := strings.Repeat(" ", margin)

	var name string
	if sel {
		name = lipgloss.NewStyle().Foreground(clrAccent).Bold(true).Render(r.secName)
	} else {
		name = lipgloss.NewStyle().Foreground(clrDim).Bold(true).Render(r.secName)
	}

	count := lipgloss.NewStyle().Foreground(clrFaint).Render(fmt.Sprintf("%d", r.count))
	countPlain := fmt.Sprintf("%d", r.count)

	usedW := margin + len(r.secName) + len(countPlain) + margin
	gap := imax(1, w-usedW)

	return pad + name + strings.Repeat(" ", gap) + count + " "
}

func (m Model) renderItem(item *model.Item, sel bool, w int) string {
	pad := strings.Repeat(" ", margin)
	depthPad := strings.Repeat("  ", item.Depth)

	// cursor column: 1 char (▶ or space)
	var cur string
	if sel {
		cur = lipgloss.NewStyle().Foreground(clrAccent).Bold(true).Render("▶")
	} else {
		cur = " "
	}

	// checkbox
	var check string
	if item.Done {
		check = lipgloss.NewStyle().Foreground(clrDone).Render("✓")
	} else {
		check = lipgloss.NewStyle().Foreground(clrFaint).Render("○")
	}

	// meta parts (plain for width calc)
	var metaParts []string
	for _, t := range item.Tags {
		metaParts = append(metaParts, t)
	}
	if item.Deadline != "" {
		metaParts = append(metaParts, shortDate(item.Deadline))
	}
	if item.Trigger != "" {
		metaParts = append(metaParts, "w:"+item.Trigger)
	}
	metaStr := strings.Join(metaParts, "  ")

	urgentStr := ""
	if item.Urgent {
		urgentStr = "  !"
	}

	// width math (plain bytes, ASCII-safe)
	// prefix = margin + depthPad + cur(1) + " " + check(1) + "  "
	prefixW := margin + len(depthPad) + 1 + 1 + 1 + 2
	rightW := 0
	if metaStr != "" {
		rightW += 2 + len(metaStr) // "  " separator
	}
	rightW += len(urgentStr)
	rightW += margin

	textAvail := w - prefixW - rightW
	if textAvail < 4 {
		textAvail = 4
	}

	text := item.Text
	if len(text) > textAvail {
		text = text[:textAvail-1] + "…"
	}
	gapW := imax(0, textAvail-len(text))

	// styled text
	var textSt string
	if item.Done {
		textSt = lipgloss.NewStyle().Foreground(clrDone).Render(text)
	} else if sel {
		textSt = lipgloss.NewStyle().Foreground(clrText).Bold(true).Render(text)
	} else {
		textSt = lipgloss.NewStyle().Foreground(clrText).Render(text)
	}

	// styled meta
	metaSt := ""
	if metaStr != "" {
		metaSt = "  " + lipgloss.NewStyle().Foreground(clrDim).Render(metaStr)
	}
	urgentSt := ""
	if item.Urgent {
		urgentSt = "  " + lipgloss.NewStyle().Foreground(clrUrgent).Render("!")
	}

	return pad + depthPad + cur + " " + check + "  " + textSt +
		strings.Repeat(" ", gapW) + metaSt + urgentSt
}

func (m Model) renderFooter(w int) string {
	pad := strings.Repeat(" ", margin)

	switch m.mode {
	case modeAdd:
		label := lipgloss.NewStyle().Foreground(clrAccent).Render(m.addSection)
		sep := lipgloss.NewStyle().Foreground(clrDim).Render("  ›  ")
		m.input.Width = w - lipgloss.Width(label) - lipgloss.Width(sep) - margin - 1
		return pad + label + sep + m.input.View()

	case modeEdit:
		label := lipgloss.NewStyle().Foreground(clrAccent).Render("edit")
		sep := lipgloss.NewStyle().Foreground(clrDim).Render("  ›  ")
		m.input.Width = w - lipgloss.Width(label) - lipgloss.Width(sep) - margin - 1
		return pad + label + sep + m.input.View()

	case modeMove:
		var parts []string
		parts = append(parts, lipgloss.NewStyle().Foreground(clrAccent).Render("move to"))
		for i, s := range m.sections {
			if i >= 9 {
				break
			}
			n := lipgloss.NewStyle().Foreground(clrDim).Render(fmt.Sprintf("%d", i+1))
			name := lipgloss.NewStyle().Foreground(clrText).Render(s.Name)
			parts = append(parts, n+name)
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(clrDim).Render("esc"))
		return pad + strings.Join(parts, "  ")

	case modeConfirmDelete:
		item := m.currentItem(m.visibleRows())
		if item == nil {
			return ""
		}
		preview := item.Text
		if len(preview) > 40 {
			preview = preview[:39] + "…"
		}
		return pad +
			lipgloss.NewStyle().Foreground(clrRed).Render("delete") +
			lipgloss.NewStyle().Foreground(clrDim).Render(` "`+preview+`"? `) +
			lipgloss.NewStyle().Foreground(clrGreen).Render("y") +
			lipgloss.NewStyle().Foreground(clrDim).Render(" yes  n cancel")
	}

	if m.err != "" {
		return pad + lipgloss.NewStyle().Foreground(clrRed).Render("✗ "+m.err)
	}

	doneHint := "hide done"
	if !m.showDone {
		doneHint = "show done"
	}
	type hint struct{ k, v string }
	var hints []hint
	if m.filterSection != "" {
		hints = []hint{
			{"j k", "nav"}, {"space", "toggle"}, {"a", "add"},
			{"e", "edit"}, {"d", "del"}, {"m", "move"}, {"h", doneHint}, {"esc", "all"}, {"q", "quit"},
		}
	} else {
		hints = []hint{
			{"j k", "nav"}, {"space", "toggle"}, {"a", "add"},
			{"e", "edit"}, {"f", "filter"}, {"d", "del"}, {"m", "move"}, {"h", doneHint}, {"q", "quit"},
		}
	}

	dot := lipgloss.NewStyle().Foreground(clrFaint).Render("·")
	var parts []string
	for _, h := range hints {
		key := lipgloss.NewStyle().Foreground(clrAccent).Render(h.k)
		val := lipgloss.NewStyle().Foreground(clrDim).Render(" " + h.v)
		parts = append(parts, key+val)
	}
	return pad + strings.Join(parts, "  "+dot+"  ")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func shortDate(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return d
	}
	return t.Format("Jan 2")
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
