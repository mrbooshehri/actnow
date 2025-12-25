package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wordwrap"

	"github.com/mrbooshehri/actNow/internal/engine"
	"github.com/mrbooshehri/actNow/internal/model"
	"github.com/mrbooshehri/actNow/internal/store"
)

type mode int

const (
	modeList mode = iota
	modeForm
	modeHelp
)

type formKind int

const (
	formAdd formKind = iota
	formEdit
)

type Model struct {
	mode              mode
	prevMode          mode
	formKind          formKind
	focusIndex        int
	store             *store.Store
	tasks             []model.Task
	selected          int
	quadrant          int
	statusMsg         string
	statusIsErr       bool
	editTaskID        string
	lastSaveTime      time.Time
	width             int
	height            int
	important         bool
	urgent            bool
	status            string
	duePicker         duePicker
	plannedPicker     duePicker
	titleInput        textinput.Model
	impactInput       textinput.Model
	nextActionInput   textinput.Model
	delegateInput     textinput.Model
	deleteReasonInput textinput.Model
	effortInput       textinput.Model
	helpOffset        int
	formEditing       bool
}

type formField int

const (
	fieldStatus formField = iota
	fieldTitle
	fieldImportant
	fieldUrgent
	fieldDue
	fieldImpact
	fieldNextAction
	fieldPlanned
	fieldEffort
	fieldDelegate
	fieldDeleteReason
)

type duePicker struct {
	enabled bool
	t       time.Time
	segment int
}

var statusOptions = []string{
	model.StatusPending,
	model.StatusDone,
	model.StatusDeferred,
}

func New(store *store.Store, tasks []model.Task) Model {
	m := Model{
		mode:     modeList,
		store:    store,
		tasks:    tasks,
		selected: 0,
		quadrant: 0,
	}
	return m
}

func (m *Model) SetStatus(msg string, isErr bool) {
	m.statusMsg = msg
	m.statusIsErr = isErr
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.applyUrgency()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.helpOffset = 0
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeForm:
			return m.updateForm(msg)
		case modeHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.mode {
	case modeList:
		return m.viewList()
	case modeForm:
		return m.viewOverlayForm()
	case modeHelp:
		return m.viewHelp()
	default:
		return ""
	}
}

func (m *Model) applyUrgency() {
	now := time.Now()
	for i := range m.tasks {
		m.tasks[i] = engine.ApplyUrgency(m.tasks[i], now)
	}
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleIndices()

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "h":
		m.prevMode = m.mode
		m.mode = modeHelp
		m.helpOffset = 0
		return m, nil
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(visible)-1 {
			m.selected++
		}
	case "tab":
		m.quadrant = (m.quadrant + 1) % 4
		m.selected = 0
	case "shift+tab":
		m.quadrant = (m.quadrant + 3) % 4
		m.selected = 0
	case "a":
		m.startForm(formAdd, model.Task{})
		return m, m.focusCmd()
	case "e":
		if len(visible) == 0 {
			return m, nil
		}
		idx := visible[m.selected]
		m.startForm(formEdit, m.tasks[idx])
		return m, m.focusCmd()
	case "d":
		if len(visible) == 0 {
			return m, nil
		}
		idx := visible[m.selected]
		if m.tasks[idx].Status == model.StatusDone {
			m.tasks[idx].Status = model.StatusPending
		} else {
			m.tasks[idx].Status = model.StatusDone
		}
		m.saveTasks()
	case "x":
		if len(visible) == 0 {
			return m, nil
		}
		idx := visible[m.selected]
		m.tasks = append(m.tasks[:idx], m.tasks[idx+1:]...)
		if m.selected > 0 && m.selected >= len(visible)-1 {
			m.selected--
		}
		m.saveTasks()
	}

	return m, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fields := m.formFields()
	if len(fields) == 0 {
		return m, nil
	}
	if m.focusIndex >= len(fields) || m.focusIndex < 0 {
		m.focusIndex = 0
	}
	current := fields[m.focusIndex]

	if m.formEditing {
		switch msg.String() {
		case "esc":
			m.formEditing = false
			return m, m.focusCmd()
		}
		if input := m.inputFor(current); input != nil {
			*input, _ = input.Update(msg)
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "ctrl+c":
		if m.formEditing {
			m.formEditing = false
			return m, m.focusCmd()
		}
		m.mode = modeList
		return m, nil
	case "up", "k", "shift+tab":
		m.focusIndex--
		if m.focusIndex < 0 {
			m.focusIndex = len(fields) - 1
		}
		return m, m.focusCmd()
	case "down", "j", "tab":
		m.focusIndex++
		if m.focusIndex >= len(fields) {
			m.focusIndex = 0
		}
		return m, m.focusCmd()
	case "enter":
		if m.focusIndex >= len(fields)-1 {
			return m.submitForm(), nil
		}
		m.focusIndex++
		return m, m.focusCmd()
	case "i":
		if m.isTextField(current) {
			m.formEditing = true
			return m, m.focusCmd()
		}
	case " ":
		if current == fieldImportant {
			m.important = !m.important
			m.focusIndex = m.indexOfField(fieldImportant)
			return m, nil
		}
		if current == fieldUrgent {
			m.urgent = !m.urgent
			m.focusIndex = m.indexOfField(fieldUrgent)
			return m, nil
		}
		if current == fieldStatus {
			m.cycleStatus(1)
			return m, nil
		}
	}

	if current == fieldDue {
		if m.handleDatePicker(&m.duePicker, msg.String()) {
			return m, nil
		}
	}

	if current == fieldPlanned {
		if m.handleDatePicker(&m.plannedPicker, msg.String()) {
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		if m.prevMode == 0 {
			m.mode = modeList
		} else {
			m.mode = m.prevMode
		}
		return m, nil
	case "up", "k":
		m.helpOffset--
	case "down", "j":
		m.helpOffset++
	case "pgup":
		m.helpOffset -= 5
	case "pgdown":
		m.helpOffset += 5
	}
	m.helpOffset = clamp(m.helpOffset, 0, m.maxHelpOffset())
	return m, nil
}

func (m *Model) startForm(kind formKind, task model.Task) {
	m.mode = modeForm
	m.formKind = kind
	m.editTaskID = task.ID
	m.focusIndex = 0
	m.formEditing = false

	m.titleInput = newInput("Title", task.Title)
	m.impactInput = newInput("Impact", task.Impact)
	m.nextActionInput = newInput("Next Action", task.NextAction)
	m.delegateInput = newInput("Delegate To", task.DelegateTo)
	m.deleteReasonInput = newInput("Delete Reason", task.DeleteReason)
	m.effortInput = newInput("Effort Estimate", task.EffortEstimate)
	if kind == formAdd {
		m.important = true
		m.urgent = true
		m.status = model.StatusPending
	} else {
		m.important = task.Important
		m.urgent = task.Urgent
		if task.Status == "" {
			m.status = model.StatusPending
		} else {
			m.status = task.Status
		}
	}
	m.duePicker = newDuePicker(task.DueAt)
	m.plannedPicker = newDuePicker(task.PlannedDate)
	m.focusIndex = m.indexOfField(fieldTitle)
}

func newInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.CharLimit = 200
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	return ti
}

func (m *Model) focusCmd() tea.Cmd {
	for _, input := range m.allInputs() {
		input.Blur()
	}
	if field := m.currentField(); field != nil {
		if m.formEditing {
			if input := m.inputFor(*field); input != nil {
				input.Focus()
			}
		} else if input := m.inputFor(*field); input != nil {
			input.Focus()
		}
	}
	return nil
}

func (m Model) submitForm() tea.Model {
	title := strings.TrimSpace(m.titleInput.Value())
	desc := ""
	var due *time.Time
	if m.duePicker.enabled {
		due = &m.duePicker.t
	}
	var planned *time.Time
	if m.plannedPicker.enabled {
		planned = &m.plannedPicker.t
	}

	if title == "" {
		m.setStatusErr("Title is required")
		return m
	}

	switch m.formKind {
	case formAdd:
		task := model.NewTask(title, desc, m.important, m.urgent, due)
		task.Status = m.statusOrDefault()
		task.Impact = strings.TrimSpace(m.impactInput.Value())
		task.NextAction = strings.TrimSpace(m.nextActionInput.Value())
		task.PlannedDate = planned
		task.DelegateTo = strings.TrimSpace(m.delegateInput.Value())
		task.DeleteReason = strings.TrimSpace(m.deleteReasonInput.Value())
		task.EffortEstimate = strings.TrimSpace(m.effortInput.Value())
		m.tasks = append(m.tasks, task)
	case formEdit:
		for i := range m.tasks {
			if m.tasks[i].ID == m.editTaskID {
				m.tasks[i].Title = title
				m.tasks[i].Important = m.important
				m.tasks[i].Urgent = m.urgent
				m.tasks[i].DueAt = due
				m.tasks[i].Status = m.statusOrDefault()
				m.tasks[i].Impact = strings.TrimSpace(m.impactInput.Value())
				m.tasks[i].NextAction = strings.TrimSpace(m.nextActionInput.Value())
				m.tasks[i].PlannedDate = planned
				m.tasks[i].DelegateTo = strings.TrimSpace(m.delegateInput.Value())
				m.tasks[i].DeleteReason = strings.TrimSpace(m.deleteReasonInput.Value())
				m.tasks[i].EffortEstimate = strings.TrimSpace(m.effortInput.Value())
				break
			}
		}
	}

	m.saveTasks()
	m.mode = modeList
	return m
}

func (m *Model) saveTasks() {
	data, err := store.EncodeTasks(m.tasks)
	if err != nil {
		m.setStatusErr("Failed to encode tasks")
		return
	}
	if err := m.store.Save(data); err != nil {
		m.setStatusErr("Failed to save tasks")
		return
	}
	m.lastSaveTime = time.Now()
}

func (m *Model) setStatusErr(msg string) {
	m.statusMsg = msg
	m.statusIsErr = true
}

func (m Model) visibleIndices() []int {
	return m.indicesByQuadrant(m.quadrant)
}

func (m Model) indicesByQuadrant(q int) []int {
	indices := make([]int, 0, len(m.tasks))
	for i, t := range m.tasks {
		if engine.QuadrantIndex(t) == q {
			indices = append(indices, i)
		}
	}
	if q == m.quadrant && m.selected >= len(indices) {
		m.selected = 0
	}
	return indices
}

func (m Model) viewList() string {
	quadrants := []string{
		engine.QuadrantImportantImmediate,
		engine.QuadrantImportantNotImmediate,
		engine.QuadrantNotImportantImmediate,
		engine.QuadrantNotImportantNot,
	}
	footer := "[↑/↓ or j/k] Move  [a] Add  [e] Edit  [d] Done  [x] Delete  [tab] Next Quadrant  [shift+tab] Prev  [h] Help  [q] Quit"

	screenW := m.width
	screenH := m.height
	if screenW == 0 || screenH == 0 {
		screenW = 80
		screenH = 24
	}
	boxGap := 1
	boxW := (screenW - boxGap) / 2
	if boxW < 10 {
		boxW = 10
	}
	if 2*boxW+boxGap > screenW {
		boxW = (screenW - boxGap) / 2
		if boxW < 1 {
			boxW = 1
		}
	}
	footerLines := 1
	available := screenH - footerLines
	boxH := available / 2
	if boxH < 5 {
		boxH = 5
	}
	if 2*boxH+footerLines > screenH {
		boxH = (screenH - footerLines) / 2
		if boxH < 3 {
			boxH = 3
		}
	}

	border := lipgloss.RoundedBorder()
	selectedBorderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	selectedTextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229"))

	boxes := make([]string, 4)
	for q := 0; q < 4; q++ {
		indices := m.indicesByQuadrant(q)
		lines := make([]string, 0, len(indices)+1)
		title := strings.ToUpper(quadrants[q])
		if len(indices) == 0 {
			lines = append(lines, "(no tasks)")
		} else {
			for i, idx := range indices {
				task := m.tasks[idx]
				cursor := " "
				if q == m.quadrant && i == m.selected {
					cursor = ">"
				}
				statusMark := "[ ]"
				switch task.Status {
				case model.StatusDone:
					statusMark = "[x]"
				case model.StatusDeferred:
					statusMark = "[-]"
				}
				due := ""
				if task.DueAt != nil {
					due = " (due " + task.DueAt.Format("2006-01-02 15:04") + ")"
				}
				lines = append(lines, fmt.Sprintf("%s %s %s%s", cursor, statusMark, task.Title, due))
			}
		}
		content := strings.Join(lines, "\n")
		borderColor, textColor := quadrantColors(q)
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		textStyle := lipgloss.NewStyle().Foreground(textColor)
		if q == m.quadrant {
			borderStyle = selectedBorderStyle
			textStyle = selectedTextStyle
		}
		maxLines := boxH - 2
		if maxLines < 1 {
			maxLines = 1
		}
		start := 0
		if len(lines) > maxLines && q == m.quadrant {
			start = clamp(m.selected-maxLines/2, 0, max(0, len(lines)-maxLines))
		}
		visibleLines := lines
		if len(lines) > maxLines {
			end := start + maxLines
			if end > len(lines) {
				end = len(lines)
			}
			visibleLines = lines[start:end]
		}
		content = strings.Join(visibleLines, "\n")
		boxes[q] = renderPanelBox(border, borderStyle, textStyle, boxW, boxH, title, content)
	}

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, boxes[0], strings.Repeat(" ", boxGap), boxes[1])
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, boxes[2], strings.Repeat(" ", boxGap), boxes[3])
	grid := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
	return lipgloss.JoinVertical(lipgloss.Left, grid, footer)
}

func quadrantColors(q int) (lipgloss.Color, lipgloss.Color) {
	switch q {
	case 0:
		return lipgloss.Color("196"), lipgloss.Color("255")
	case 1:
		return lipgloss.Color("160"), lipgloss.Color("255")
	case 2:
		return lipgloss.Color("220"), lipgloss.Color("0")
	default:
		return lipgloss.Color("27"), lipgloss.Color("255")
	}
}

func renderPanelBox(border lipgloss.Border, borderStyle, textStyle lipgloss.Style, width, height int, title, content string) string {
	if width < 2 || height < 2 {
		return content
	}
	innerWidth := width - 2
	innerHeight := height - 2

	titleRunes := []rune(title)
	if len(titleRunes) > innerWidth-2 {
		titleRunes = titleRunes[:innerWidth-2]
	}

	var b strings.Builder
	b.WriteString(renderTopBorder(border, borderStyle, innerWidth, titleRunes))
	b.WriteString("\n")

	contentLines := strings.Split(content, "\n")
	for i := 0; i < innerHeight; i++ {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}
		line = fitLine(line, innerWidth)
		b.WriteString(borderStyle.Render(border.Left))
		b.WriteString(textStyle.Render(line))
		b.WriteString(borderStyle.Render(border.Right))
		if i < innerHeight-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(borderStyle.Render(border.BottomLeft))
	b.WriteString(borderStyle.Render(strings.Repeat(border.Bottom, innerWidth)))
	b.WriteString(borderStyle.Render(border.BottomRight))

	return b.String()
}

func renderTopBorder(border lipgloss.Border, borderStyle lipgloss.Style, innerWidth int, title []rune) string {
	if innerWidth < 1 {
		return borderStyle.Render(border.TopLeft + border.TopRight)
	}
	inner := make([]rune, 0, innerWidth)
	if len(title) > 0 && innerWidth >= 2 {
		inner = append(inner, ' ')
		inner = append(inner, title...)
		inner = append(inner, ' ')
	}
	for len(inner) < innerWidth {
		inner = append(inner, []rune(border.Top)...)
		if len(inner) > innerWidth {
			inner = inner[:innerWidth]
		}
	}
	return borderStyle.Render(border.TopLeft) + borderStyle.Render(string(inner)) + borderStyle.Render(border.TopRight)
}

func fitLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	line = truncate.String(line, uint(width))
	visible := ansi.PrintableRuneWidth(line)
	if visible >= width {
		return line
	}
	return line + strings.Repeat(" ", width-visible)
}

func checkbox(checked bool) string {
	if checked {
		return "[x]"
	}
	return "[ ]"
}

func (m Model) formLine(field formField, label, value string) string {
	cursor := " "
	if current := m.currentField(); current != nil && *current == field {
		cursor = ">"
	}
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	return fmt.Sprintf("%s %s: %s", cursor, labelStyle.Render(label), value)
}

func (m Model) formFields() []formField {
	switch {
	case m.important && m.urgent:
		return []formField{fieldStatus, fieldTitle, fieldImportant, fieldUrgent, fieldDue, fieldImpact, fieldNextAction}
	case m.important:
		return []formField{fieldStatus, fieldTitle, fieldImportant, fieldUrgent, fieldPlanned, fieldEffort}
	case m.urgent:
		return []formField{fieldStatus, fieldTitle, fieldImportant, fieldUrgent, fieldDue, fieldDelegate}
	default:
		return []formField{fieldTitle, fieldImportant, fieldUrgent, fieldDeleteReason}
	}
}

func (m Model) isTextField(field formField) bool {
	switch field {
	case fieldTitle, fieldImpact, fieldNextAction, fieldDelegate, fieldDeleteReason, fieldEffort:
		return true
	default:
		return false
	}
}

func (m Model) indexOfField(target formField) int {
	fields := m.formFields()
	for i, field := range fields {
		if field == target {
			return i
		}
	}
	return 0
}

func (m Model) currentField() *formField {
	fields := m.formFields()
	if len(fields) == 0 {
		return nil
	}
	if m.focusIndex >= len(fields) {
		return &fields[0]
	}
	return &fields[m.focusIndex]
}

func (m Model) formFieldLine(field formField) string {
	switch field {
	case fieldStatus:
		return m.formLine(fieldStatus, "Status", m.statusDisplay())
	case fieldTitle:
		return m.formLine(fieldTitle, "Title", m.titleInput.View())
	case fieldImportant:
		return m.formLine(fieldImportant, "Important", checkbox(m.important))
	case fieldUrgent:
		return m.formLine(fieldUrgent, "Urgent", checkbox(m.urgent))
	case fieldDue:
		return m.formLine(fieldDue, "Due/SLA", m.duePicker.String())
	case fieldImpact:
		return m.formLine(fieldImpact, "Impact", m.impactInput.View())
	case fieldNextAction:
		return m.formLine(fieldNextAction, "Next Action", m.nextActionInput.View())
	case fieldPlanned:
		return m.formLine(fieldPlanned, "Planned Date", m.plannedPicker.String())
	case fieldEffort:
		return m.formLine(fieldEffort, "Effort", m.effortInput.View())
	case fieldDelegate:
		return m.formLine(fieldDelegate, "Delegate To", m.delegateInput.View())
	case fieldDeleteReason:
		return m.formLine(fieldDeleteReason, "Delete Reason", m.deleteReasonInput.View())
	default:
		return ""
	}
}

func (m Model) statusDisplay() string {
	return "[" + m.statusOrDefault() + "]"
}

func (m Model) statusOrDefault() string {
	if m.status == "" {
		return model.StatusPending
	}
	return m.status
}

func (m *Model) cycleStatus(delta int) {
	current := m.statusOrDefault()
	index := 0
	for i, opt := range statusOptions {
		if opt == current {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = len(statusOptions) - 1
	}
	if index >= len(statusOptions) {
		index = 0
	}
	m.status = statusOptions[index]
}

func (m Model) allInputs() []*textinput.Model {
	return []*textinput.Model{
		&m.titleInput,
		&m.impactInput,
		&m.nextActionInput,
		&m.delegateInput,
		&m.deleteReasonInput,
		&m.effortInput,
	}
}

func (m *Model) inputFor(field formField) *textinput.Model {
	switch field {
	case fieldTitle:
		return &m.titleInput
	case fieldImpact:
		return &m.impactInput
	case fieldNextAction:
		return &m.nextActionInput
	case fieldDelegate:
		return &m.delegateInput
	case fieldDeleteReason:
		return &m.deleteReasonInput
	case fieldEffort:
		return &m.effortInput
	default:
		return nil
	}
}

func newDuePicker(due *time.Time) duePicker {
	if due == nil {
		return duePicker{
			enabled: false,
			t:       time.Now().Truncate(time.Minute),
			segment: 0,
		}
	}
	return duePicker{
		enabled: true,
		t:       due.Truncate(time.Minute),
		segment: 0,
	}
}

func (p duePicker) String() string {
	if !p.enabled {
		return "(empty)"
	}
	hi := lipgloss.NewStyle().Underline(true).Bold(true)
	year, month, day := p.t.Date()
	hour, min, _ := p.t.Clock()

	segments := []string{
		fmt.Sprintf("%04d", year),
		fmt.Sprintf("%02d", int(month)),
		fmt.Sprintf("%02d", day),
		fmt.Sprintf("%02d", hour),
		fmt.Sprintf("%02d", min),
	}
	if p.segment >= 0 && p.segment < len(segments) {
		segments[p.segment] = hi.Render(segments[p.segment])
	}
	return fmt.Sprintf("%s-%s-%s %s:%s", segments[0], segments[1], segments[2], segments[3], segments[4])
}

func (m *Model) handleDatePicker(p *duePicker, key string) bool {
	switch key {
	case "x":
		p.enabled = false
		return true
	case "t":
		p.enabled = true
		p.t = time.Now().Truncate(time.Minute)
		return true
	case "left", "h":
		p.segment--
		if p.segment < 0 {
			p.segment = 4
		}
		return true
	case "right", "l":
		p.segment++
		if p.segment > 4 {
			p.segment = 0
		}
		return true
	case "+":
		p.enabled = true
		p.adjust(1)
		return true
	case "-":
		p.enabled = true
		p.adjust(-1)
		return true
	default:
		return false
	}
}

func (p *duePicker) adjust(delta int) {
	switch p.segment {
	case 0:
		p.t = p.t.AddDate(delta, 0, 0)
	case 1:
		p.t = p.t.AddDate(0, delta, 0)
	case 2:
		p.t = p.t.AddDate(0, 0, delta)
	case 3:
		p.t = p.t.Add(time.Duration(delta) * time.Hour)
	case 4:
		p.t = p.t.Add(time.Duration(delta) * time.Minute)
	}
}

func (m Model) viewOverlayForm() string {
	base := m.viewList()
	modal := m.viewModalBox()
	return overlayCenter(base, modal, m.width, m.height)
}

func (m Model) viewModalBox() string {
	title := "Add Task"
	if m.formKind == formEdit {
		title = "Edit Task"
	}

	width := m.width
	height := m.height
	if width == 0 || height == 0 {
		width = 80
		height = 24
	}
	boxW := width * 2 / 3
	if boxW < 50 {
		boxW = 50
	}
	if boxW > width-4 {
		boxW = width - 4
	}
	boxH := height / 2
	if boxH < 10 {
		boxH = 10
	}

	fields := m.formFields()
	lines := make([]string, 0, len(fields)+2)
	for _, field := range fields {
		lines = append(lines, m.formFieldLine(field))
	}
	lines = append(lines, "[enter] Next  [esc] Cancel")
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	hintText := "[↑/↓, j/k]: move fields  [i]: insert  [enter]: next/save  [esc]: exit/close  [space]: toggle  date: h/l segment  +/- change  t current time  x clear"
	hintLines := []string{hintText}
	innerHeight := boxH - 2
	if innerHeight > 0 {
		hintLines = flattenWrapped([]string{wrapLine(hintText, boxW-2)})
		for i := range hintLines {
			hintLines[i] = hintStyle.Render(hintLines[i])
		}
		need := len(hintLines)
		if need == 0 {
			need = 1
			hintLines = []string{hintStyle.Render("")}
		}
		if len(lines)+need > innerHeight {
			lines = lines[:max(0, innerHeight-need)]
		}
		for len(lines) < innerHeight-need {
			lines = append(lines, "")
		}
		lines = append(lines, hintLines...)
	}

	border := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	return renderPanelBox(border, borderStyle, textStyle, boxW, boxH, title, content)
}

func overlayCenter(base, modal string, width, height int) string {
	if width == 0 || height == 0 {
		return base
	}
	baseLines := padToSize(base, width, height)
	modalLines := strings.Split(modal, "\n")
	modalW := maxLineWidth(modalLines)
	modalH := len(modalLines)
	if modalW == 0 || modalH == 0 {
		return base
	}
	for i := range modalLines {
		modalLines[i] = padLine(modalLines[i], modalW)
	}

	startX := (width - modalW) / 2
	if startX < 0 {
		startX = 0
	}
	startY := (height - modalH) / 2
	if startY < 0 {
		startY = 0
	}

	for i := 0; i < modalH && startY+i < len(baseLines); i++ {
		left, right := splitByWidth(baseLines[startY+i], startX, modalW)
		baseLines[startY+i] = left + modalLines[i] + right
	}
	return strings.Join(baseLines, "\n")
}

func (m Model) viewHelp() string {
	width := m.width
	height := m.height
	if width == 0 || height == 0 {
		width = 80
		height = 24
	}

	header := "HELP — Eisenhower (ActNow)"
	footer := "[↑/↓, j/k] scroll  [h/esc/q] back"
	usableHeight := height - 2
	if usableHeight < 1 {
		usableHeight = 1
	}

	wrapped := m.helpLines(width)

	offset := clamp(m.helpOffset, 0, max(0, len(wrapped)-usableHeight))
	end := offset + usableHeight
	if end > len(wrapped) {
		end = len(wrapped)
	}
	view := wrapped[offset:end]
	for len(view) < usableHeight {
		view = append(view, "")
	}

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	var b strings.Builder
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	for i := 0; i < usableHeight-1; i++ {
		if i < len(view) {
			b.WriteString(bodyStyle.Render(view[i]))
		}
		if i < usableHeight-2 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(footer))

	return padToScreen(b.String(), width, height)
}

func padToSize(s string, width, height int) []string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = padLine(lines[i], width)
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return lines
}

func padToScreen(s string, width, height int) string {
	lines := padToSize(s, width, height)
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func (m Model) helpLines(width int) []string {
	lines := []string{
		"Important–Immediate (Urgent) Quadrant Method",
		"It helps you decide what to do now, later, delegate, or ignore.",
		"",
		"Ask two questions:",
		"1) Is it Important? Does it matter for goals, money, safety, or health?",
		"2) Is it Immediate (Urgent)? Does it need action now or very soon?",
		"",
		"The 4 Quadrants",
		"IMPORTANT + IMMEDIATE = Do Now",
		"IMPORTANT + NOT IMMEDIATE = Plan",
		"NOT IMPORTANT + IMMEDIATE = Delegate",
		"NOT IMPORTANT + NOT IMMEDIATE = Eliminate",
		"",
		"1) Do Now (Important & Immediate)",
		"- Deadline or emergency; must be done now",
		"Examples: prod outage, critical bug, bill due today, health emergency",
		"How: do immediately, don't delay, don't delegate unless necessary",
		"",
		"2) Plan (Important & Not Immediate)",
		"- Important long-term, but not urgent yet",
		"Examples: learning, documentation, architecture, training, savings",
		"How: schedule it; if ignored, it becomes urgent later",
		"",
		"3) Delegate (Not Important & Immediate)",
		"- Feels urgent but low impact; interrupts focus",
		"Examples: random calls, non-critical emails, low-value meetings",
		"How: delegate or do quickly; protect your important work",
		"",
		"4) Eliminate (Not Important & Not Immediate)",
		"- No value and no urgency",
		"Examples: endless scrolling, random videos, gossip",
		"How: avoid or limit; don't schedule it",
		"",
		"Simple workday example (DevOps)",
		"- Production outage: Do Now",
		"- Learning Helm: Plan",
		"- Non-critical emails: Delegate",
		"- Scrolling Twitter: Eliminate",
		"",
		"Golden Rule for Beginners",
		"- Survive: focus on Quadrant 1",
		"- Succeed: invest time in Quadrant 2",
		"- Protect time: minimize Quadrant 3",
		"- Waste less life: eliminate Quadrant 4",
		"",
		"Summary: Do urgent-important now, plan important early, delegate urgent distractions, remove useless tasks.",
		"",
		"Navigation",
		"- [↑/↓] or k/j: move within a quadrant",
		"- [tab]: switch quadrant",
		"- [a]: add task, [e]: edit task, [d]: mark done, [x]: delete, [q]: quit",
		"",
		"Quadrants",
		"- I+I (Important & Immediate): status, title, due/SLA, impact, next action",
		"- I+NI (Important & Not Immediate): status, title, planned date, effort",
		"- NI+I (Not Important & Immediate): status, title, due/SLA, delegate to",
		"- NI+NI (Not Important & Not Immediate): title, delete reason",
		"",
		"Form editing",
		"- [↑/↓] or j/k: move fields, [i] insert, [enter] next/save",
		"- [esc] exit insert or close the form",
		"- [space]: toggle checkboxes",
		"- Date fields: [h/l] move segment, [j/k] change value, [t] now, [x] clear",
		"",
		"Examples",
		"1) I+I incident",
		"   Title: Fix prod outage",
		"   Impact: Revenue loss",
		"   Next Action: Restart database",
		"   Due/SLA: 2025-01-05 13:00",
		"",
		"2) I+NI plan",
		"   Title: Write migration plan",
		"   Planned Date: 2025-01-12 09:00",
		"   Effort: 4h",
		"",
		"3) NI+I quick task",
		"   Title: Renew SSL cert",
		"   Delegate To: ops@team",
		"   Due/SLA: 2025-01-07 10:00",
		"",
		"4) NI+NI cleanup",
		"   Title: Remove old test data",
		"   Delete Reason: Not needed",
	}

	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, wrapLine(line, width))
	}
	return flattenWrapped(wrapped)
}

func (m Model) maxHelpOffset() int {
	width := m.width
	height := m.height
	if width == 0 || height == 0 {
		width = 80
		height = 24
	}
	usableHeight := height - 2
	if usableHeight < 1 {
		usableHeight = 1
	}
	lines := m.helpLines(width)
	if len(lines) <= usableHeight {
		return 0
	}
	return len(lines) - usableHeight
}

func padLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	s = truncate.String(s, uint(width))
	visible := ansi.PrintableRuneWidth(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func maxLineWidth(lines []string) int {
	max := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > max {
			max = w
		}
	}
	return max
}

func splitByWidth(s string, start, width int) (string, string) {
	if start < 0 {
		start = 0
	}
	if width < 0 {
		width = 0
	}
	var left, right strings.Builder
	visible := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			seq, n := readANSI(s[i:])
			if visible < start {
				left.WriteString(seq)
			} else if visible >= start+width {
				right.WriteString(seq)
			}
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			if visible < start {
				left.WriteByte(s[i])
			} else if visible >= start+width {
				right.WriteByte(s[i])
			}
			visible++
			i++
			continue
		}
		rw := runewidth.RuneWidth(r)
		if rw < 0 {
			rw = 0
		}
		if visible+rw <= start {
			left.WriteString(string(r))
		} else if visible >= start+width {
			right.WriteString(string(r))
		}
		visible += rw
		i += size
	}
	return left.String(), right.String()
}

func wrapLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	return wordwrap.String(s, width)
}

func flattenWrapped(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "\n") {
			out = append(out, strings.Split(line, "\n")...)
		} else {
			out = append(out, line)
		}
	}
	return out
}

func clamp(v, minVal, maxVal int) int {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func readANSI(s string) (string, int) {
	if len(s) == 0 || s[0] != '\x1b' {
		return "", 0
	}
	i := 1
	if i < len(s) && s[i] == '[' {
		i++
		for i < len(s) {
			c := s[i]
			i++
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				break
			}
		}
		return s[:i], i
	}
	return s[:1], 1
}
