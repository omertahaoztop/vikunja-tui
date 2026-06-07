package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/omertahaoztop/vikunja-tui/internal/vikunja"
)

type board struct {
	projectTitle string
	buckets      []vikunja.Bucket
	selBucket    int
	selTask      int
	filter       string
}

func newBoard(title string) board {
	return board{projectTitle: title}
}

// visibleTasks returns the tasks of a bucket after applying the active filter.
func (b board) visibleTasks(bucketIdx int) []vikunja.Task {
	if bucketIdx < 0 || bucketIdx >= len(b.buckets) {
		return nil
	}
	all := b.buckets[bucketIdx].Tasks
	if b.filter == "" {
		return all
	}
	needle := strings.ToLower(b.filter)
	var out []vikunja.Task
	for _, t := range all {
		if strings.Contains(strings.ToLower(t.Title), needle) {
			out = append(out, t)
		}
	}
	return out
}

func (b *board) update(buckets []vikunja.Bucket) {
	// Drop nil-title buckets (Vikunja sometimes returns placeholder entries).
	var kept []vikunja.Bucket
	for _, bk := range buckets {
		if strings.TrimSpace(bk.Title) != "" {
			kept = append(kept, bk)
		}
	}
	b.buckets = kept
	if b.selBucket >= len(b.buckets) {
		b.selBucket = max(0, len(b.buckets)-1)
	}
	b.clampTask()
}

func (b *board) clampTask() {
	n := len(b.visibleTasks(b.selBucket))
	if n == 0 {
		b.selTask = 0
		return
	}
	if b.selTask >= n {
		b.selTask = n - 1
	}
	if b.selTask < 0 {
		b.selTask = 0
	}
}

func (b *board) nextBucket() {
	if len(b.buckets) == 0 {
		return
	}
	b.selBucket = (b.selBucket + 1) % len(b.buckets)
	b.selTask = 0
}

func (b *board) prevBucket() {
	if len(b.buckets) == 0 {
		return
	}
	b.selBucket = (b.selBucket - 1 + len(b.buckets)) % len(b.buckets)
	b.selTask = 0
}

func (b *board) nextTask() {
	n := len(b.visibleTasks(b.selBucket))
	if n == 0 {
		return
	}
	b.selTask = (b.selTask + 1) % n
}

func (b *board) prevTask() {
	n := len(b.visibleTasks(b.selBucket))
	if n == 0 {
		return
	}
	b.selTask = (b.selTask - 1 + n) % n
}

func (b board) currentBucket() (vikunja.Bucket, bool) {
	if b.selBucket < 0 || b.selBucket >= len(b.buckets) {
		return vikunja.Bucket{}, false
	}
	return b.buckets[b.selBucket], true
}

func (b board) selectedTask() (vikunja.Task, bool) {
	tasks := b.visibleTasks(b.selBucket)
	if b.selTask < 0 || b.selTask >= len(tasks) {
		return vikunja.Task{}, false
	}
	return tasks[b.selTask], true
}

// findBucketIndex returns the slice index of the bucket with the given id.
func (b board) findBucketIndex(id int64) (int, bool) {
	for i, bk := range b.buckets {
		if bk.ID == id {
			return i, true
		}
	}
	return -1, false
}

func renderBadges(th Theme, t vikunja.Task, width int) string {
	var parts []string
	if p := t.PriorityLabel(); p != "" {
		parts = append(parts, th.BadgePrio.Render(p))
	}
	if t.HasDue() {
		parts = append(parts, th.BadgeDue.Render("⏱ "+humanDue(t.DueDate)))
	}
	for _, l := range t.Labels {
		parts = append(parts, th.BadgeLabel.Render("#"+l.Title))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func humanDue(d time.Time) string {
	now := time.Now()
	days := int(d.Sub(now).Hours() / 24)
	switch {
	case days < 0:
		return "overdue"
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	default:
		return fmt.Sprintf("%dd", days)
	}
}

func (b board) view(th Theme, width, height int, loading bool) string {
	if loading && len(b.buckets) == 0 {
		msg := th.ModalBody.Render("● Loading board...")
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
	}
	if len(b.buckets) == 0 {
		msg := th.ModalBody.Render("No buckets in this project.")
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
	}

	n := len(b.buckets)
	gap := 1
	colW := (width - (n-1)*gap) / n
	if colW < 18 {
		colW = 18
	}
	if colW > 42 {
		colW = 42
	}
	innerW := colW - 4
	if innerW < 8 {
		innerW = 8
	}

	var cols []string
	for i, bk := range b.buckets {
		tasks := b.visibleTasks(i)
		focused := i == b.selBucket

		hdrStyle := th.ColHeader
		if focused {
			hdrStyle = th.ColHeaderF
		}
		header := hdrStyle.Width(innerW).Render(
			truncate(bk.Title, innerW-5) + fmt.Sprintf(" [%d]", len(tasks)),
		)

		var taskLines []string
		taskLines = append(taskLines, header, "")
		if len(tasks) == 0 {
			taskLines = append(taskLines, th.Card.Render("  —"))
		}
		for j, t := range tasks {
			bullet := "· "
			cardStyle := th.Card
			if t.Done {
				cardStyle = th.CardDone
			}
			if focused && j == b.selTask {
				bullet = "▶ "
				cardStyle = th.CardFocused
			}
			title := truncate(t.DisplayTitle(), innerW-2)
			taskLines = append(taskLines, cardStyle.Width(innerW).Render(bullet+title))
			if badges := renderBadges(th, t, innerW); badges != "" {
				taskLines = append(taskLines, "  "+truncate(badges, innerW-2))
			}
		}

		maxRows := height - 2
		if maxRows > 0 && len(taskLines) > maxRows {
			taskLines = taskLines[:maxRows]
		}

		body := strings.Join(taskLines, "\n")
		colStyle := th.Column
		if focused {
			colStyle = th.ColumnFoc
		}
		cols = append(cols, colStyle.Width(colW).Height(height-2).Render(body))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

func truncate(s string, max int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if max <= 1 || len(r) <= max {
		return string(r)
	}
	return string(r[:max-1]) + "…"
}
