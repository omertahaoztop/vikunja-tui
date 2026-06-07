package vikunja

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/omertahaoztop/vikunja-tui/internal/config"
)

type Client struct {
	base   string
	token  string
	http   *http.Client
	meName string
}

func ensureAPIV1(url string) string {
	clean := strings.TrimRight(url, "/")
	switch {
	case strings.HasSuffix(clean, "/api/v1"):
		return clean
	case strings.HasSuffix(clean, "/api"):
		return clean + "/v1"
	default:
		return clean + "/api/v1"
	}
}

func New(base, token string) *Client {
	return &Client{
		base:  ensureAPIV1(base),
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

func FromConfig(cfg *config.Config) (*Client, error) {
	if cfg.APIToken != "" {
		return New(cfg.APIURL, cfg.APIToken), nil
	}
	return login(cfg.APIURL, cfg.Username, cfg.Password)
}

func login(base, username, password string) (*Client, error) {
	url := ensureAPIV1(base) + "/login"
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("login failed (%d): %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("login: bad response: %w", err)
	}
	if out.Token == "" {
		return nil, fmt.Errorf("login failed: no token in response")
	}
	return New(base, out.Token), nil
}

func (c *Client) request(method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("vikunja API %s %s -> %d: %s", method, path, resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return nil
}

func (c *Client) get(path string, out any) error { return c.request(http.MethodGet, path, nil, out) }
func (c *Client) post(path string, payload, out any) error {
	return c.request(http.MethodPost, path, payload, out)
}
func (c *Client) put(path string, payload, out any) error {
	return c.request(http.MethodPut, path, payload, out)
}
func (c *Client) del(path string) error { return c.request(http.MethodDelete, path, nil, nil) }

func (c *Client) MeName() string {
	if c.meName != "" {
		return c.meName
	}
	var u struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := c.get("/user", &u); err != nil {
		c.meName = "Unknown"
		return c.meName
	}
	switch {
	case u.Name != "":
		c.meName = u.Name
	case u.Username != "":
		c.meName = u.Username
	default:
		c.meName = "Unknown"
	}
	return c.meName
}

func (c *Client) Projects() ([]Project, error) {
	var projects []Project
	if err := c.get("/projects", &projects); err != nil {
		return nil, err
	}
	out := make([]Project, 0, len(projects))
	for _, p := range projects {
		if !p.IsArchived {
			out = append(out, p)
		}
	}
	return out, nil
}
