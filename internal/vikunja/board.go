package vikunja

import "fmt"

func (c *Client) findKanbanView(p *Project) (*View, error) {
	views := p.Views
	if len(views) == 0 {
		if err := c.get(fmt.Sprintf("/projects/%d/views", p.ID), &views); err != nil {
			return nil, err
		}
	}
	for i := range views {
		if views[i].ViewKind == "kanban" {
			return &views[i], nil
		}
	}
	if len(views) > 0 {
		return &views[0], nil
	}
	return nil, nil
}

// LoadBoard returns kanban buckets (with nested tasks) and stores view metadata
// on the project pointer for later task creation / done-bucket moves.
func (c *Client) LoadBoard(p *Project) ([]Bucket, error) {
	view, err := c.findKanbanView(p)
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, nil
	}

	p.ViewID = view.ID
	p.DoneBucketID = view.DoneBucketID
	p.DefaultBucketID = view.DefaultBucketID

	var buckets []Bucket
	path := fmt.Sprintf("/projects/%d/views/%d/tasks", p.ID, view.ID)
	if err := c.get(path, &buckets); err != nil {
		return nil, err
	}
	return buckets, nil
}

func (c *Client) CreateTask(p *Project, bucketID int64, title string) (Task, error) {
	var created Task
	payload := map[string]any{"title": title, "bucket_id": bucketID}
	if err := c.put(fmt.Sprintf("/projects/%d/tasks", p.ID), payload, &created); err != nil {
		return Task{}, err
	}
	if created.ID != 0 && p.ViewID != 0 {
		_ = c.post(
			fmt.Sprintf("/projects/%d/views/%d/buckets/%d", p.ID, p.ViewID, bucketID),
			map[string]any{"task_id": created.ID},
			nil,
		)
	}
	return created, nil
}

func (c *Client) UpdateTask(t Task) (Task, error) {
	var updated Task
	if err := c.post(fmt.Sprintf("/tasks/%d", t.ID), t, &updated); err != nil {
		return Task{}, err
	}
	return updated, nil
}

func (c *Client) DeleteTask(id int64) error {
	return c.del(fmt.Sprintf("/tasks/%d", id))
}
