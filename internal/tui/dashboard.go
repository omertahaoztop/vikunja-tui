package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/omertahaoztop/vikunja-tui/internal/vikunja"
)

type treeRow struct {
	project vikunja.Project
	depth   int
	hasKids bool
}

type dashboard struct {
	rows     []treeRow
	selected int
}

func newDashboard() dashboard {
	return dashboard{}
}

func (d *dashboard) populate(projects []vikunja.Project) {
	childrenOf := map[int64][]vikunja.Project{}
	var roots []vikunja.Project
	for _, p := range projects {
		if p.ParentProjectID > 0 {
			childrenOf[p.ParentProjectID] = append(childrenOf[p.ParentProjectID], p)
		} else {
			roots = append(roots, p)
		}
	}

	var rows []treeRow
	var walk func(p vikunja.Project, depth int)
	walk = func(p vikunja.Project, depth int) {
		kids := childrenOf[p.ID]
		rows = append(rows, treeRow{project: p, depth: depth, hasKids: len(kids) > 0})
		for _, c := range kids {
			walk(c, depth+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}

	d.rows = rows
	if d.selected >= len(rows) {
		d.selected = 0
	}
}

func (d *dashboard) next() {
	if len(d.rows) == 0 {
		return
	}
	d.selected = (d.selected + 1) % len(d.rows)
}

func (d *dashboard) prev() {
	if len(d.rows) == 0 {
		return
	}
	d.selected = (d.selected - 1 + len(d.rows)) % len(d.rows)
}

func (d *dashboard) selectedProject() (vikunja.Project, bool) {
	if d.selected < 0 || d.selected >= len(d.rows) {
		return vikunja.Project{}, false
	}
	return d.rows[d.selected].project, true
}

func (d dashboard) view(th Theme, width, height int, loading bool) string {
	logo := th.Logo.Render("vikunja · tui")

	if loading && len(d.rows) == 0 {
		body := th.ModalBody.Render("● Loading projects...")
		content := lipgloss.JoinVertical(lipgloss.Center, logo, "", body)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
	}

	if len(d.rows) == 0 {
		body := th.ModalBody.Render("No projects found.")
		content := lipgloss.JoinVertical(lipgloss.Center, logo, "", body)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
	}

	var lines []string
	for i, r := range d.rows {
		indent := strings.Repeat("  ", r.depth)
		marker := "·"
		if r.hasKids {
			marker = "▸"
		}
		label := indent + marker + " " + r.project.DisplayTitle()
		if i == d.selected {
			lines = append(lines, th.TreeSelected.Render(" "+label+" "))
		} else {
			lines = append(lines, th.TreeItem.Render(" "+label+" "))
		}
	}

	panel := th.Column.Width(60).Render(strings.Join(lines, "\n"))
	content := lipgloss.JoinVertical(lipgloss.Center, logo, "", panel)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
