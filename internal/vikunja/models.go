package vikunja

import (
	"strings"
	"time"
)

type Label struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	HexColor string `json:"hex_color"`
}

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	BucketID    int64     `json:"bucket_id"`
	Priority    int       `json:"priority"`
	DueDate     time.Time `json:"due_date"`
	Labels      []Label   `json:"labels"`
	Identifier  string    `json:"identifier"`
	Index       int64     `json:"index"`
}

func (t Task) DisplayTitle() string {
	if s := strings.TrimSpace(t.Title); s != "" {
		return s
	}
	return "Untitled"
}

func (t Task) HasDue() bool {
	return !t.DueDate.IsZero() && t.DueDate.Year() > 1
}

func (t Task) PriorityLabel() string {
	switch {
	case t.Priority >= 5:
		return "DO NOW"
	case t.Priority == 4:
		return "URGENT"
	case t.Priority == 3:
		return "HIGH"
	case t.Priority == 2:
		return "MED"
	case t.Priority == 1:
		return "LOW"
	default:
		return ""
	}
}

type Bucket struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
	Tasks []Task `json:"tasks"`
}

type View struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	ViewKind        string `json:"view_kind"`
	DoneBucketID    int64  `json:"done_bucket_id"`
	DefaultBucketID int64  `json:"default_bucket_id"`
}

type Project struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	ParentProjectID int64  `json:"parent_project_id"`
	IsArchived      bool   `json:"is_archived"`
	Views           []View `json:"views"`

	ViewID          int64 `json:"-"`
	DoneBucketID    int64 `json:"-"`
	DefaultBucketID int64 `json:"-"`
}

func (p Project) DisplayTitle() string {
	if s := strings.TrimSpace(p.Title); s != "" {
		return s
	}
	return "Untitled"
}
