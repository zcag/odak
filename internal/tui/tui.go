package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zcag/odak/internal/client"
	"github.com/zcag/odak/internal/model"
)

var (
	reTrail = regexp.MustCompile(`\S+$`)
	reACTok = regexp.MustCompile(`(?:^|\s)([#/]|d:)(\S*)$`)
)

// chip is a parsed token attached to the add/edit input.
type chip struct{ kind, value string }

type acItem struct{ label, value string }
type acState struct {
	mode  string
	items []acItem
}

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
	includeTags   []string
	excludeTags   []string
	mode          modeID
	input         textinput.Model
	addSection    string
	editID        string
	addChips      []chip
	acIndex       int
	acDismissed   bool
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
	return Model{cl: cl, input: ti, loading: true, showDone: false}
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
	case modeAdd, modeEdit:
		return m.handleInputKey(k)
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
		} else if len(m.includeTags) > 0 || len(m.excludeTags) > 0 {
			m.includeTags = nil
			m.excludeTags = nil
			m = m.clampScroll()
		}

	case "w":
		m = m.cycleTag("work")
		m = m.clampScroll()

	case "p":
		m = m.cycleTag("personal")
		m = m.clampScroll()

	case "c":
		if len(m.includeTags) > 0 || len(m.excludeTags) > 0 {
			m.includeTags = nil
			m.excludeTags = nil
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
		m.input.Placeholder = "new item…   #tag  /section  d:date  ! urgent"
		m.input.SetValue("")
		m.addChips = nil
		m.acIndex = 0
		m.acDismissed = false
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
			m.addChips = chipsFromItem(item)
			m.acIndex = 0
			m.acDismissed = false
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

func (m Model) handleInputKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	ks := k.String()
	ac := m.currentAC()

	// autocomplete navigation has priority
	if ac != nil {
		switch ks {
		case "down":
			m.acIndex = (m.acIndex + 1) % len(ac.items)
			return m, nil
		case "up":
			m.acIndex = (m.acIndex - 1 + len(ac.items)) % len(ac.items)
			return m, nil
		case "tab":
			it := ac.items[m.acIndex%len(ac.items)]
			m.commitChip(ac.mode, it.value)
			return m, nil
		}
	}

	switch ks {
	case "esc":
		if ac != nil {
			m.acDismissed = true
			return m, nil
		}
		m.mode = modeBrowse
		m.editID = ""
		m.addChips = nil
		m.acDismissed = false
		m.input.Blur()
		return m, nil

	case "enter":
		if ac != nil {
			it := ac.items[m.acIndex%len(ac.items)]
			m.commitChip(ac.mode, it.value)
			return m, nil
		}
		return m.submitInput()

	case " ":
		if m.maybeCommitTrailing() {
			return m, nil
		}

	case "backspace":
		if m.input.Value() == "" && len(m.addChips) > 0 {
			m.addChips = m.addChips[:len(m.addChips)-1]
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(k)
	return m, cmd
}

func (m Model) submitInput() (tea.Model, tea.Cmd) {
	m.maybeCommitTrailing()
	text := strings.TrimSpace(m.input.Value())

	chips := m.addChips
	isEdit := m.mode == modeEdit
	editID := m.editID
	defaultSec := m.addSection

	m.mode = modeBrowse
	m.addChips = nil
	m.editID = ""
	m.acDismissed = false
	m.input.Blur()

	if text == "" {
		return m, nil
	}

	var tags []string
	var urgent bool
	var deadline, section string
	for _, c := range chips {
		switch c.kind {
		case "tag":
			tags = append(tags, c.value)
		case "section":
			section = c.value
		case "deadline":
			deadline = c.value
		case "urgent":
			urgent = true
		}
	}

	if isEdit {
		return m, m.cmdEdit(editID, text, tags, urgent, deadline, section)
	}
	if section == "" {
		section = defaultSec
	}
	return m, m.cmdAdd(text, tags, urgent, deadline, section)
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
			if item.Section != sec.Name {
				continue
			}
			if !m.showDone && item.Done {
				continue
			}
			if !m.matchesTagFilter(item) {
				continue
			}
			rows = append(rows, row{kind: rowItem, item: item})
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

func (m Model) cmdAdd(text string, tags []string, urgent bool, deadline, section string) tea.Cmd {
	it := &model.Item{Text: text, Section: section, Tags: tags, Urgent: urgent, Deadline: deadline}
	return func() tea.Msg {
		if _, err := m.cl.Create(it); err != nil {
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

func (m Model) cmdEdit(id, text string, tags []string, urgent bool, deadline, section string) tea.Cmd {
	if tags == nil {
		tags = []string{}
	}
	patch := &client.ItemPatch{Text: text, Tags: tags, Urgent: urgent, Deadline: deadline}
	return func() tea.Msg {
		if _, err := m.cl.Patch(id, patch); err != nil {
			return errMsg{err}
		}
		if section != "" {
			if _, err := m.cl.Move(id, section); err != nil {
				return errMsg{err}
			}
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

// ── chip + autocomplete + tag filter helpers ──────────────────────────────────

func chipsFromItem(it *model.Item) []chip {
	var out []chip
	for _, t := range it.Tags {
		out = append(out, chip{kind: "tag", value: t})
	}
	if it.Urgent {
		out = append(out, chip{kind: "urgent"})
	}
	if it.Deadline != "" {
		out = append(out, chip{kind: "deadline", value: it.Deadline})
	}
	return out
}

func (m Model) hasChip(kind, value string) bool {
	for _, c := range m.addChips {
		if c.kind == kind && (value == "" || c.value == value) {
			return true
		}
	}
	return false
}

func (m *Model) commitChip(kind, value string) {
	text := reTrail.ReplaceAllString(m.input.Value(), "")
	m.input.SetValue(text)
	m.input.CursorEnd()
	switch kind {
	case "urgent":
		if !m.hasChip("urgent", "") {
			m.addChips = append(m.addChips, chip{kind: kind})
		}
	case "section", "deadline":
		filtered := m.addChips[:0]
		for _, c := range m.addChips {
			if c.kind != kind {
				filtered = append(filtered, c)
			}
		}
		m.addChips = append(filtered, chip{kind: kind, value: value})
	case "tag":
		if !m.hasChip("tag", value) {
			m.addChips = append(m.addChips, chip{kind: kind, value: value})
		}
	}
	m.acIndex = 0
	m.acDismissed = false
}

// maybeCommitTrailing tries to convert the current trailing token into a chip.
// Returns true if a chip was committed.
func (m *Model) maybeCommitTrailing() bool {
	trailing := reTrail.FindString(m.input.Value())
	if trailing == "" {
		return false
	}
	switch {
	case trailing == "!":
		m.commitChip("urgent", "")
		return true
	case strings.HasPrefix(trailing, "#") && len(trailing) > 1:
		m.commitChip("tag", trailing[1:])
		return true
	case strings.HasPrefix(trailing, "d:") && len(trailing) > 2:
		m.commitChip("deadline", trailing[2:])
		return true
	case strings.HasPrefix(trailing, "/") && len(trailing) > 1:
		q := strings.ToLower(trailing[1:])
		for _, sec := range m.sections {
			if strings.HasPrefix(strings.ToLower(sec.Name), q) {
				m.commitChip("section", sec.Name)
				return true
			}
		}
	}
	return false
}

func (m Model) collectTags() []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range m.allItems {
		for _, t := range it.Tags {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

func ymd(offset int) string {
	return time.Now().AddDate(0, 0, offset).Format("2006-01-02")
}

func (m Model) currentAC() *acState {
	if m.acDismissed {
		return nil
	}
	matches := reACTok.FindStringSubmatch(m.input.Value())
	if matches == nil {
		return nil
	}
	trig, q := matches[1], strings.ToLower(matches[2])
	s := &acState{}
	switch trig {
	case "#":
		s.mode = "tag"
		for _, t := range m.collectTags() {
			if strings.Contains(strings.ToLower(t), q) && !m.hasChip("tag", t) {
				s.items = append(s.items, acItem{label: "#" + t, value: t})
			}
		}
	case "/":
		s.mode = "section"
		for _, sec := range m.sections {
			if strings.Contains(strings.ToLower(sec.Name), q) {
				s.items = append(s.items, acItem{label: sec.Name, value: sec.Name})
			}
		}
	case "d:":
		s.mode = "deadline"
		if q != "" {
			s.items = append(s.items, acItem{label: q, value: q})
		}
		s.items = append(s.items,
			acItem{label: "today", value: ymd(0)},
			acItem{label: "tomorrow", value: ymd(1)},
			acItem{label: "in a week", value: ymd(7)},
		)
	}
	if len(s.items) == 0 {
		return nil
	}
	return s
}

func (m Model) cycleTag(tag string) Model {
	inIdx, exIdx := -1, -1
	for i, t := range m.includeTags {
		if t == tag {
			inIdx = i
			break
		}
	}
	for i, t := range m.excludeTags {
		if t == tag {
			exIdx = i
			break
		}
	}
	switch {
	case inIdx >= 0:
		m.includeTags = append(m.includeTags[:inIdx], m.includeTags[inIdx+1:]...)
		m.excludeTags = append(m.excludeTags, tag)
	case exIdx >= 0:
		m.excludeTags = append(m.excludeTags[:exIdx], m.excludeTags[exIdx+1:]...)
	default:
		m.includeTags = append(m.includeTags, tag)
	}
	return m
}

func (m Model) matchesTagFilter(it *model.Item) bool {
	if len(m.includeTags) > 0 {
		ok := false
		for _, t := range it.Tags {
			for _, inc := range m.includeTags {
				if t == inc {
					ok = true
					break
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, t := range it.Tags {
		for _, ex := range m.excludeTags {
			if t == ex {
				return false
			}
		}
	}
	return true
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
	base := m.h - 4 // header + top-div + bot-div + footer
	if (m.mode == modeAdd || m.mode == modeEdit) && m.currentAC() != nil {
		base--
	}
	if base < 1 {
		base = 1
	}
	return base
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
	left := strings.Repeat(" ", margin) + brand
	if m.filterSection != "" {
		sep := lipgloss.NewStyle().Foreground(clrDim).Render("  /  ")
		sec := lipgloss.NewStyle().Foreground(clrText).Render(m.filterSection)
		left += sep + sec
	}
	if len(m.includeTags) > 0 || len(m.excludeTags) > 0 {
		dot := lipgloss.NewStyle().Foreground(clrFaint).Render("  ·  ")
		var parts []string
		for _, t := range m.includeTags {
			parts = append(parts, lipgloss.NewStyle().Foreground(clrAccent).Render("+"+t))
		}
		for _, t := range m.excludeTags {
			parts = append(parts, lipgloss.NewStyle().Foreground(clrDim).Strikethrough(true).Render(t))
		}
		left += dot + strings.Join(parts, " ")
	}

	open := 0
	for _, it := range m.allItems {
		if !it.Done && m.matchesTagFilter(it) {
			if m.filterSection != "" && it.Section != m.filterSection {
				continue
			}
			open++
		}
	}
	summary := fmt.Sprintf("%d open", open)
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
	case modeAdd, modeEdit:
		labelTxt := m.addSection
		if m.mode == modeEdit {
			labelTxt = "edit"
		}
		label := lipgloss.NewStyle().Foreground(clrAccent).Render(labelTxt)
		sep := lipgloss.NewStyle().Foreground(clrDim).Render("  ›  ")

		var chipParts []string
		for _, c := range m.addChips {
			chipParts = append(chipParts, m.renderChip(c))
		}
		chipsStr := ""
		if len(chipParts) > 0 {
			chipsStr = strings.Join(chipParts, " ") + " "
		}

		prefix := pad + label + sep + chipsStr
		m.input.Width = w - lipgloss.Width(prefix) - margin - 1
		if m.input.Width < 10 {
			m.input.Width = 10
		}
		inputLine := prefix + m.input.View()

		ac := m.currentAC()
		if ac == nil {
			return inputLine
		}
		acLine := pad + m.renderAC(ac, w)
		return acLine + "\n" + inputLine

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
	hints = append(hints, hint{"j k", "nav"}, hint{"space", "toggle"}, hint{"a", "add"}, hint{"e", "edit"})
	if m.filterSection == "" {
		hints = append(hints, hint{"f", "filter"})
	}
	hints = append(hints, hint{"d", "del"}, hint{"m", "move"})
	hints = append(hints, hint{"w/p", "tag"})
	if len(m.includeTags) > 0 || len(m.excludeTags) > 0 {
		hints = append(hints, hint{"c", "clear"})
	}
	hints = append(hints, hint{"h", doneHint})
	if m.filterSection != "" || len(m.includeTags) > 0 || len(m.excludeTags) > 0 {
		hints = append(hints, hint{"esc", "back"})
	}
	hints = append(hints, hint{"q", "quit"})

	dot := lipgloss.NewStyle().Foreground(clrFaint).Render("·")
	var parts []string
	for _, h := range hints {
		key := lipgloss.NewStyle().Foreground(clrAccent).Render(h.k)
		val := lipgloss.NewStyle().Foreground(clrDim).Render(" " + h.v)
		parts = append(parts, key+val)
	}
	return pad + strings.Join(parts, "  "+dot+"  ")
}

func (m Model) renderChip(c chip) string {
	switch c.kind {
	case "tag":
		return lipgloss.NewStyle().Foreground(clrAccent).Render("#" + c.value)
	case "section":
		return lipgloss.NewStyle().Foreground(clrAccent).Render("/" + c.value)
	case "deadline":
		return lipgloss.NewStyle().Foreground(clrDim).Render("d:" + shortDate(c.value))
	case "urgent":
		return lipgloss.NewStyle().Foreground(clrUrgent).Bold(true).Render("!")
	}
	return ""
}

func (m Model) renderAC(ac *acState, w int) string {
	max := 6
	if len(ac.items) < max {
		max = len(ac.items)
	}
	var parts []string
	for i := 0; i < max; i++ {
		it := ac.items[i]
		if i == m.acIndex {
			parts = append(parts, lipgloss.NewStyle().Foreground(clrAccent).Bold(true).Render(it.label))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(clrDim).Render(it.label))
		}
	}
	more := ""
	if len(ac.items) > max {
		more = lipgloss.NewStyle().Foreground(clrFaint).Render(fmt.Sprintf("  +%d", len(ac.items)-max))
	}
	hint := lipgloss.NewStyle().Foreground(clrFaint).Render("  tab")
	return strings.Join(parts, "  ") + more + hint
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
