package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type modalKind int

const (
	modalNone modalKind = iota
	modalInput
	modalConfirm
	modalDetail
	modalEdit
)

// pending describes the action a confirm modal will run on "yes".
type pendingKind int

const (
	pendingNone pendingKind = iota
	pendingDeleteTask
	pendingClearBucket
)

type pending struct {
	kind    pendingKind
	taskID  int64
	taskIDs []int64
}

type modal struct {
	kind modalKind

	title   string
	body    string
	input   textinput.Model
	desc    textarea.Model
	editID  int64
	confYes bool
	pend    pending
	scroll  int
}

func newInputModal(prompt string) modal {
	ti := textinput.New()
	ti.SetWidth(48)
	ti.Focus()
	return modal{kind: modalInput, title: prompt, input: ti}
}

func newConfirmModal(prompt string, p pending) modal {
	return modal{kind: modalConfirm, title: prompt, confYes: false, pend: p}
}

func newDetailModal(title, body string) modal {
	if strings.TrimSpace(body) == "" {
		body = "No description."
	}
	return modal{kind: modalDetail, title: title, body: body}
}

func newEditModal(t titleDesc) modal {
	ti := textinput.New()
	ti.SetWidth(48)
	ti.SetValue(t.Title)
	ti.Focus()

	ta := textarea.New()
	ta.SetWidth(48)
	ta.SetHeight(6)
	ta.SetValue(t.Description)

	return modal{kind: modalEdit, title: "Edit task", input: ti, desc: ta, editID: t.ID}
}

type titleDesc struct {
	ID          int64
	Title       string
	Description string
}

func (m modal) update(msg tea.Msg) (modal, tea.Cmd) {
	var cmd tea.Cmd
	switch m.kind {
	case modalInput:
		m.input, cmd = m.input.Update(msg)
	case modalEdit:
		// Tab toggles between title and description focus.
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "tab" {
			if m.input.Focused() {
				m.input.Blur()
				cmd = m.desc.Focus()
			} else {
				m.desc.Blur()
				cmd = m.input.Focus()
			}
			return m, cmd
		}
		if m.input.Focused() {
			m.input, cmd = m.input.Update(msg)
		} else {
			m.desc, cmd = m.desc.Update(msg)
		}
	}
	return m, cmd
}

func (m modal) view(th Theme, width, height int) string {
	var inner string
	switch m.kind {
	case modalInput:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			th.ModalTitle.Render(m.title),
			"",
			m.input.View(),
			"",
			th.ModalBody.Render("Enter: OK   Esc: Cancel"),
		)
	case modalConfirm:
		yes := " Yes [y] "
		no := " No [n/esc] "
		yesStyle := lipgloss.NewStyle().Foreground(th.P.FgSecond)
		noStyle := lipgloss.NewStyle().Foreground(th.P.FgSecond)
		if m.confYes {
			yesStyle = lipgloss.NewStyle().Foreground(th.P.BgPrimary).Background(th.P.Red).Bold(true)
		} else {
			noStyle = lipgloss.NewStyle().Foreground(th.P.BgPrimary).Background(th.P.Accent).Bold(true)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Center,
			yesStyle.Render(yes), "  ", noStyle.Render(no))
		inner = lipgloss.JoinVertical(lipgloss.Center,
			th.ModalTitle.Render(m.title), "", row)
	case modalDetail:
		body := m.body
		lines := strings.Split(body, "\n")
		maxBody := height - 8
		if maxBody < 1 {
			maxBody = 1
		}
		if m.scroll > 0 && m.scroll < len(lines) {
			lines = lines[m.scroll:]
		}
		if len(lines) > maxBody {
			lines = lines[:maxBody]
		}
		inner = lipgloss.JoinVertical(lipgloss.Left,
			th.ModalTitle.Render(m.title),
			"",
			th.ModalBody.Render(strings.Join(lines, "\n")),
			"",
			th.ModalBody.Render("↑↓: Scroll   Esc/q: Close"),
		)
	case modalEdit:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			th.ModalTitle.Render(m.title),
			"",
			th.ModalBody.Render("Title:"),
			m.input.View(),
			"",
			th.ModalBody.Render("Description:"),
			m.desc.View(),
			"",
			th.ModalBody.Render("Tab: Switch   Enter: Save   Esc: Cancel"),
		)
	default:
		return ""
	}

	box := th.Modal.Width(54).Render(inner)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
