package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mrbooshehri/actNow/internal/engine"
	"github.com/mrbooshehri/actNow/internal/model"
	"github.com/mrbooshehri/actNow/internal/store"
)

type mode int

const (
	modeList mode = iota
	modeForm
)

type formKind int

const (
	formAdd formKind = iota
	formEdit
)

type Model struct {
	mode         mode
	formKind     formKind
	inputs       []textinput.Model
	focusIndex   int
	store        *store.Store
	tasks        []model.Task
	selected     int
	quadrant     int
	statusMsg    string
	statusIsErr  bool
	editTaskID   string
	lastSaveTime time.Time
	width        int
	height       int
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
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeForm:
			return m.updateForm(msg)
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.mode {
	case modeList:
		return m.viewList()
	case modeForm:
		return m.viewForm()
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
		m.tasks[idx].Status = model.StatusDone
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
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeList
		m.statusMsg = "Canceled"
		m.statusIsErr = false
		return m, nil
	case "enter":
		if m.focusIndex >= len(m.inputs)-1 {
			return m.submitForm(), nil
		}
		m.focusIndex++
		return m, m.focusCmd()
	}

	for i := range m.inputs {
		m.inputs[i], _ = m.inputs[i].Update(msg)
	}
	return m, nil
}

func (m *Model) startForm(kind formKind, task model.Task) {
	m.mode = modeForm
	m.formKind = kind
	m.editTaskID = task.ID
	m.inputs = make([]textinput.Model, 5)
	m.focusIndex = 0

	m.inputs[0] = newInput("Title", task.Title)
	m.inputs[1] = newInput("Description", task.Description)
	m.inputs[2] = newInput("Important (y/n)", boolToYN(task.Important, kind == formAdd))
	m.inputs[3] = newInput("Urgent (y/n)", boolToYN(task.Urgent, kind == formAdd))
	m.inputs[4] = newInput("Due (YYYY-MM-DD HH:MM or empty)", formatDue(task.DueAt))
}

func boolToYN(v bool, defaultYes bool) string {
	if v {
		return "y"
	}
	if defaultYes {
		return "y"
	}
	return "n"
}

func formatDue(due *time.Time) string {
	if due == nil {
		return ""
	}
	return due.Format("2006-01-02 15:04")
}

func newInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.CharLimit = 200
	return ti
}

func (m Model) focusCmd() tea.Cmd {
	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()
	}
	return nil
}

func (m Model) submitForm() tea.Model {
	title := strings.TrimSpace(m.inputs[0].Value())
	desc := strings.TrimSpace(m.inputs[1].Value())
	important, err := parseBool(m.inputs[2].Value())
	if err != nil {
		m.setStatusErr("Important must be y or n")
		return m
	}
	urgent, err := parseBool(m.inputs[3].Value())
	if err != nil {
		m.setStatusErr("Urgent must be y or n")
		return m
	}
	var due *time.Time
	if strings.TrimSpace(m.inputs[4].Value()) != "" {
		parsed, err := parseDue(m.inputs[4].Value())
		if err != nil {
			m.setStatusErr("Due must be YYYY-MM-DD HH:MM")
			return m
		}
		due = &parsed
	}

	if title == "" {
		m.setStatusErr("Title is required")
		return m
	}

	switch m.formKind {
	case formAdd:
		m.tasks = append(m.tasks, model.NewTask(title, desc, important, urgent, due))
	case formEdit:
		for i := range m.tasks {
			if m.tasks[i].ID == m.editTaskID {
				m.tasks[i].Title = title
				m.tasks[i].Description = desc
				m.tasks[i].Important = important
				m.tasks[i].Urgent = urgent
				m.tasks[i].DueAt = due
				break
			}
		}
	}

	m.saveTasks()
	m.mode = modeList
	m.statusMsg = "Saved"
	m.statusIsErr = false
	return m
}

func parseBool(v string) (bool, error) {
	s := strings.TrimSpace(strings.ToLower(v))
	switch s {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid")
	}
}

func parseDue(v string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, strings.TrimSpace(v)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid")
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
	footer := "[a] Add  [e] Edit  [d] Done  [x] Delete  [tab] Next Quadrant  [q] Quit"
	status := ""
	if m.statusMsg != "" {
		prefix := "OK"
		if m.statusIsErr {
			prefix = "ERR"
		}
		status = fmt.Sprintf("%s: %s", prefix, m.statusMsg)
	}

	screenW := m.width
	screenH := m.height
	if screenW < 80 {
		screenW = 80
	}
	if screenH < 24 {
		screenH = 24
	}
	boxGap := 1
	boxW := (screenW - boxGap) / 2
	footerLines := 2
	boxH := (screenH - footerLines - 1) / 2

	baseStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252")).
		Width(boxW).
		Height(boxH)
	selectedStyle := baseStyle.Copy().
		BorderForeground(lipgloss.Color("214")).
		Foreground(lipgloss.Color("229"))

	boxes := make([]string, 4)
	for q := 0; q < 4; q++ {
		indices := m.indicesByQuadrant(q)
		lines := make([]string, 0, len(indices)+1)
		title := strings.ToUpper(quadrants[q])
		lines = append(lines, title)
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
				if task.IsDone() {
					statusMark = "[x]"
				}
				due := ""
				if task.DueAt != nil {
					due = " (due " + task.DueAt.Format("2006-01-02 15:04") + ")"
				}
				lines = append(lines, fmt.Sprintf("%s %s %s%s", cursor, statusMark, task.Title, due))
			}
		}
		content := strings.Join(lines, "\n")
		style := baseStyle
		if q == m.quadrant {
			style = selectedStyle
		}
		boxes[q] = style.Render(content)
	}

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, boxes[0], strings.Repeat(" ", boxGap), boxes[1])
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, boxes[2], strings.Repeat(" ", boxGap), boxes[3])
	grid := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
	footerBlock := footer
	if status != "" {
		footerBlock = footer + "\n" + status
	}
	return lipgloss.JoinVertical(lipgloss.Left, grid, footerBlock)
}

func (m Model) viewForm() string {
	var b strings.Builder
	if m.formKind == formAdd {
		b.WriteString("Add Task\n")
	} else {
		b.WriteString("Edit Task\n")
	}
	b.WriteString("----------------\n")
	for i, input := range m.inputs {
		cursor := " "
		if i == m.focusIndex {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s: %s\n", cursor, input.Placeholder, input.View())
	}
	b.WriteString("\n[enter] Next  [esc] Cancel\n")
	if m.statusMsg != "" {
		prefix := "OK"
		if m.statusIsErr {
			prefix = "ERR"
		}
		fmt.Fprintf(&b, "%s: %s\n", prefix, m.statusMsg)
	}
	return b.String()
}
