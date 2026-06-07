package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/omertahaoztop/vikunja-tui/internal/vikunja"
)

type screen int

const (
	screenDashboard screen = iota
	screenBoard
)

type notifLevel int

const (
	notifInfo notifLevel = iota
	notifWarn
	notifError
)

const (
	syncInterval = 5 * time.Second
	notifTTL     = 3 * time.Second
)

// ---- messages -------------------------------------------------------------

type projectsMsg struct{ projects []vikunja.Project }
type boardMsg struct{ buckets []vikunja.Bucket }
type taskCreatedMsg struct{ task vikunja.Task }
type taskUpdatedMsg struct{}
type taskDeletedMsg struct{}
type batchDeletedMsg struct{ deleted, failed int }
type errMsg struct{ err string }
type notifyMsg struct{ text string }
type tickMsg struct{}

// ---- model ----------------------------------------------------------------

type Model struct {
	client *vikunja.Client

	screen    screen
	dashboard dashboard
	board     board
	current   vikunja.Project

	modal     *modal
	search    bool
	searchBuf string
	help      bool

	themeIdx int
	theme    Theme

	width, height int
	loading       bool

	notif      string
	notifLvl   notifLevel
	notifUntil time.Time

	quit bool
}

func New(client *vikunja.Client) Model {
	return Model{
		client:    client,
		screen:    screenDashboard,
		dashboard: newDashboard(),
		board:     newBoard(""),
		theme:     NewTheme(Palettes[0]),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadProjects(), tick())
}

// ---- commands -------------------------------------------------------------

func tick() tea.Cmd {
	return tea.Tick(syncInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) loadProjects() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		projects, err := c.Projects()
		if err != nil {
			return errMsg{"Connection error: " + err.Error()}
		}
		return projectsMsg{projects}
	}
}

func (m Model) loadBoard() tea.Cmd {
	c := m.client
	p := m.current
	return func() tea.Msg {
		buckets, err := c.LoadBoard(&p)
		if err != nil {
			return errMsg{"Failed to load board: " + err.Error()}
		}
		return boardMsg{buckets}
	}
}

// syncBoard mirrors loadBoard but swallows errors (background refresh).
func (m Model) syncBoard() tea.Cmd {
	c := m.client
	p := m.current
	return func() tea.Msg {
		buckets, err := c.LoadBoard(&p)
		if err != nil {
			return nil
		}
		return boardMsg{buckets}
	}
}

func (m Model) createTask(title string) tea.Cmd {
	c := m.client
	p := m.current
	bk, ok := m.board.currentBucket()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		task, err := c.CreateTask(&p, bk.ID, title)
		if err != nil {
			return errMsg{"Create failed: " + err.Error()}
		}
		return taskCreatedMsg{task}
	}
}

func (m Model) deleteTask(id int64) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		if err := c.DeleteTask(id); err != nil {
			return errMsg{"Delete failed: " + err.Error()}
		}
		return taskDeletedMsg{}
	}
}

func (m Model) clearBucket(ids []int64) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		deleted, failed := 0, 0
		for _, id := range ids {
			if c.DeleteTask(id) == nil {
				deleted++
			} else {
				failed++
			}
		}
		return batchDeletedMsg{deleted, failed}
	}
}

func (m Model) updateTask(t vikunja.Task) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		if _, err := c.UpdateTask(t); err != nil {
			return errMsg{"Update failed: " + err.Error()}
		}
		return taskUpdatedMsg{}
	}
}

// ---- update ---------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case projectsMsg:
		m.loading = false
		m.dashboard.populate(msg.projects)
		return m, nil

	case boardMsg:
		m.loading = false
		m.board.update(msg.buckets)
		return m, nil

	case taskCreatedMsg:
		m.notify("Added: "+oneLine(msg.task.Title, 30), notifInfo)
		return m, m.loadBoard()

	case taskUpdatedMsg:
		return m, m.loadBoard()

	case taskDeletedMsg:
		m.notify("Deleted.", notifInfo)
		return m, m.loadBoard()

	case batchDeletedMsg:
		if msg.failed > 0 {
			m.notify(itoa(msg.deleted)+" deleted, "+itoa(msg.failed)+" failed.", notifWarn)
		} else {
			m.notify(itoa(msg.deleted)+" task(s) deleted.", notifInfo)
		}
		return m, m.loadBoard()

	case errMsg:
		m.loading = false
		m.notify(msg.err, notifError)
		return m, nil

	case notifyMsg:
		m.notify(msg.text, notifInfo)
		return m, nil

	case tickMsg:
		var cmd tea.Cmd
		if m.screen == screenBoard && m.modal == nil && !m.search {
			cmd = m.syncBoard()
		}
		return m, tea.Batch(cmd, tick())
	}

	// Forward to active modal (textinput/textarea updates).
	if m.modal != nil {
		um, cmd := m.modal.update(msg)
		m.modal = &um
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.modal != nil {
		return m.handleModalKey(k)
	}
	if m.search {
		return m.handleSearchKey(k)
	}
	if m.help {
		if k.String() == "?" || k.String() == "esc" || k.String() == "q" {
			m.help = false
		}
		return m, nil
	}

	switch m.screen {
	case screenDashboard:
		return m.handleDashboardKey(k)
	default:
		return m.handleBoardKey(k)
	}
}

func (m Model) handleDashboardKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "?":
		m.help = true
	case "t":
		return m.cycleTheme()
	case "j", "down":
		m.dashboard.next()
	case "k", "up":
		m.dashboard.prev()
	case "enter":
		if p, ok := m.dashboard.selectedProject(); ok {
			m.current = p
			m.screen = screenBoard
			m.board = newBoard(p.DisplayTitle())
			m.loading = true
			return m, m.loadBoard()
		}
	}
	return m, nil
}

func (m Model) handleBoardKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "esc":
		m.screen = screenDashboard
		m.board = newBoard("")
	case "?":
		m.help = true
	case "t":
		return m.cycleTheme()
	case "/":
		m.search = true
		m.searchBuf = m.board.filter
	case "tab", "right", "l":
		m.board.nextBucket()
	case "shift+tab", "left", "h":
		m.board.prevBucket()
	case "j", "down":
		m.board.nextTask()
	case "k", "up":
		m.board.prevTask()
	case "a":
		md := newInputModal("New task title:")
		m.modal = &md
	case "e":
		if t, ok := m.board.selectedTask(); ok {
			md := newEditModal(titleDesc{ID: t.ID, Title: t.Title, Description: t.Description})
			m.modal = &md
		} else {
			m.notify("No task selected.", notifWarn)
		}
	case "d":
		return m.promptDeleteTask()
	case "D":
		return m.promptClearBucket()
	case "c":
		return m.toggleDone()
	case ">":
		return m.moveTask(1)
	case "<":
		return m.moveTask(-1)
	case "enter":
		if t, ok := m.board.selectedTask(); ok {
			md := newDetailModal(t.DisplayTitle(), t.Description)
			m.modal = &md
		}
	case "r":
		m.loading = true
		return m, m.loadBoard()
	}
	return m, nil
}

func (m Model) handleSearchKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "enter", "esc":
		m.search = false
		m.board.filter = m.searchBuf
		if k.String() == "esc" {
			m.searchBuf = ""
			m.board.filter = ""
		}
		m.board.selTask = 0
	case "backspace":
		if len(m.searchBuf) > 0 {
			r := []rune(m.searchBuf)
			m.searchBuf = string(r[:len(r)-1])
			m.board.filter = m.searchBuf
		}
	default:
		if len(k.String()) == 1 {
			m.searchBuf += k.String()
			m.board.filter = m.searchBuf
			m.board.selTask = 0
		}
	}
	return m, nil
}

func (m Model) handleModalKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	md := *m.modal
	switch md.kind {
	case modalInput:
		switch k.String() {
		case "enter":
			val := strings.TrimSpace(md.input.Value())
			m.modal = nil
			if val != "" {
				return m, m.createTask(val)
			}
		case "esc":
			m.modal = nil
		default:
			um, cmd := md.update(k)
			m.modal = &um
			return m, cmd
		}
	case modalConfirm:
		switch k.String() {
		case "y", "Y":
			p := md.pend
			m.modal = nil
			return m.runPending(p)
		case "n", "N", "esc":
			m.modal = nil
		case "left", "right", "tab":
			md.confYes = !md.confYes
			m.modal = &md
		case "enter":
			yes := md.confYes
			p := md.pend
			m.modal = nil
			if yes {
				return m.runPending(p)
			}
		}
	case modalDetail:
		switch k.String() {
		case "esc", "q":
			m.modal = nil
		case "j", "down":
			md.scroll++
			m.modal = &md
		case "k", "up":
			if md.scroll > 0 {
				md.scroll--
			}
			m.modal = &md
		}
	case modalEdit:
		switch k.String() {
		case "esc":
			m.modal = nil
		case "enter":
			if md.input.Focused() {
				title := strings.TrimSpace(md.input.Value())
				desc := md.desc.Value()
				id := md.editID
				m.modal = nil
				if t, ok := m.board.selectedTask(); ok && t.ID == id {
					t.Title = title
					t.Description = desc
					return m, m.updateTask(t)
				}
				return m, nil
			}
			um, cmd := md.update(k)
			m.modal = &um
			return m, cmd
		default:
			um, cmd := md.update(k)
			m.modal = &um
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) runPending(p pending) (tea.Model, tea.Cmd) {
	switch p.kind {
	case pendingDeleteTask:
		return m, m.deleteTask(p.taskID)
	case pendingClearBucket:
		return m, m.clearBucket(p.taskIDs)
	}
	return m, nil
}

func (m Model) promptDeleteTask() (tea.Model, tea.Cmd) {
	t, ok := m.board.selectedTask()
	if !ok {
		m.notify("No task selected.", notifWarn)
		return m, nil
	}
	md := newConfirmModal("Delete '"+oneLine(t.Title, 40)+"'?", pending{kind: pendingDeleteTask, taskID: t.ID})
	m.modal = &md
	return m, nil
}

func (m Model) promptClearBucket() (tea.Model, tea.Cmd) {
	bk, ok := m.board.currentBucket()
	if !ok {
		m.notify("No bucket selected.", notifWarn)
		return m, nil
	}
	if len(bk.Tasks) == 0 {
		m.notify("Bucket is already empty.", notifWarn)
		return m, nil
	}
	ids := make([]int64, 0, len(bk.Tasks))
	for _, t := range bk.Tasks {
		ids = append(ids, t.ID)
	}
	md := newConfirmModal(
		"Delete all "+itoa(len(ids))+" tasks in '"+bk.Title+"'?",
		pending{kind: pendingClearBucket, taskIDs: ids},
	)
	m.modal = &md
	return m, nil
}

func (m Model) toggleDone() (tea.Model, tea.Cmd) {
	t, ok := m.board.selectedTask()
	if !ok {
		m.notify("No task selected.", notifWarn)
		return m, nil
	}
	t.Done = !t.Done
	if t.Done {
		m.notify("Marked as done.", notifInfo)
	} else {
		m.notify("Marked as not done.", notifInfo)
	}
	return m, m.updateTask(t)
}

func (m Model) moveTask(dir int) (tea.Model, tea.Cmd) {
	t, ok := m.board.selectedTask()
	if !ok {
		m.notify("No task selected.", notifWarn)
		return m, nil
	}
	target := m.board.selBucket + dir
	if target < 0 || target >= len(m.board.buckets) {
		return m, nil
	}
	dest := m.board.buckets[target]
	t.BucketID = dest.ID
	m.notify("Moved to "+dest.Title+".", notifInfo)
	return m, m.updateTask(t)
}

func (m Model) cycleTheme() (tea.Model, tea.Cmd) {
	m.themeIdx = (m.themeIdx + 1) % len(Palettes)
	m.theme = NewTheme(Palettes[m.themeIdx])
	m.notify("Theme: "+Palettes[m.themeIdx].Name, notifInfo)
	return m, nil
}

func (m *Model) notify(text string, lvl notifLevel) {
	m.notif = text
	m.notifLvl = lvl
	m.notifUntil = time.Now().Add(notifTTL)
}

// ---- view -----------------------------------------------------------------

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}
	th := m.theme

	header := m.renderHeader()
	footer := m.renderFooter()
	bodyH := m.height - 2
	if bodyH < 1 {
		bodyH = 1
	}

	var body string
	switch m.screen {
	case screenDashboard:
		body = m.dashboard.view(th, m.width, bodyH, m.loading)
	default:
		body = m.board.view(th, m.width, bodyH, m.loading)
	}

	out := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	switch {
	case m.help:
		out = m.overlay(out, m.renderHelp())
	case m.modal != nil:
		out = m.overlay(out, m.modal.view(th, m.width, m.height))
	case m.notif != "" && time.Now().Before(m.notifUntil):
		out = m.overlayNotif(out)
	}

	v := tea.NewView(out)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.BackgroundColor = th.P.BgPrimary
	return v
}

func (m Model) overlay(base, top string) string {
	// Top already Place()-centered to full screen, so just return it over body.
	_ = base
	return top
}

func (m Model) overlayNotif(base string) string {
	th := m.theme
	var style lipgloss.Style
	icon := "✓"
	switch m.notifLvl {
	case notifWarn:
		style, icon = th.NotifWarn, "⚠"
	case notifError:
		style, icon = th.NotifError, "✗"
	default:
		style = th.NotifInfo
	}
	notif := style.Render(icon + " " + m.notif)
	placed := lipgloss.Place(m.width, m.height, lipgloss.Right, lipgloss.Bottom, notif)
	// Composite: base lines, overwrite last region with notif via simple overlay.
	return compositeBottomRight(base, placed, m.width, m.height)
}

func (m Model) renderHeader() string {
	th := m.theme
	left := th.HeaderLogo.Render(" ⬡ vikunja-tui")
	if m.screen == screenBoard && m.board.projectTitle != "" {
		left += th.HeaderSub.Render("  ·  " + m.board.projectTitle)
	}
	status := th.StatusOK.Render("● Connected")
	if m.loading {
		status = th.StatusSync.Render("● Syncing… ")
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(status)
	if gap < 1 {
		gap = 1
	}
	filler := th.Header.Render(strings.Repeat(" ", gap))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, filler, status)
}

func (m Model) renderFooter() string {
	th := m.theme
	var crumbs, hints string
	if m.search {
		crumbs = "Search: " + m.searchBuf + "▏"
		hints = "Enter: Apply   Esc: Clear"
	} else {
		switch m.screen {
		case screenDashboard:
			crumbs = "Dashboard"
			hints = "↑↓ Navigate  ⏎ Open  t Theme  ? Help  q Quit"
		default:
			crumbs = "Dashboard › " + m.board.projectTitle
			hints = "←→ Buckets  ↑↓ Tasks  a Add  e Edit  d Del  c Done  / Search  ? Help"
		}
	}
	left := th.Footer.Render(" " + crumbs)
	right := th.KeyHint.Render(hints + " ")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	filler := th.Footer.Render(strings.Repeat(" ", gap))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, filler, right)
}

func (m Model) renderHelp() string {
	th := m.theme
	rows := []string{
		th.ModalTitle.Render("Keyboard Shortcuts"),
		"",
		"Navigation",
		"  ↑↓ / j k       Move selection",
		"  ←→ / h l / Tab Switch bucket",
		"  Enter          Open / details",
		"  Esc            Back / cancel",
		"",
		"Tasks",
		"  a   Add task        e   Edit task",
		"  d   Delete task     D   Clear bucket",
		"  c   Toggle done     < > Move bucket",
		"  /   Search filter   r   Reload board",
		"",
		"Global",
		"  t   Cycle theme     ?   This help",
		"  q   Quit",
	}
	box := th.Modal.Width(52).Render(strings.Join(rows, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// ---- helpers --------------------------------------------------------------

func compositeBottomRight(base, overlayFull string, w, h int) string {
	baseLines := strings.Split(base, "\n")
	overLines := strings.Split(overlayFull, "\n")
	for i := 0; i < len(baseLines) && i < len(overLines); i++ {
		if strings.TrimSpace(overLines[i]) != "" {
			baseLines[i] = overLines[i]
		}
	}
	return strings.Join(baseLines, "\n")
}

func oneLine(text string, max int) string {
	line := strings.TrimSpace(strings.SplitN(strings.ReplaceAll(text, "\r", ""), "\n", 2)[0])
	if line == "" {
		return "Untitled"
	}
	r := []rune(line)
	if max > 0 && len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return line
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
