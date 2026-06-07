package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	APIURL   string
	APIToken string
	Username string
	Password string
}

func searchPaths() []string {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return []string{
		"/etc/default/vikunja-tui",
		filepath.Join(home, ".config", "vikunja-tui", "config"),
		filepath.Join(cwd, ".env"),
	}
}

func Load() (*Config, error) {
	loaded := false
	for _, p := range searchPaths() {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Overload(p)
			loaded = true
			break
		}
	}
	if !loaded {
		_ = godotenv.Load()
	}

	cfg := &Config{
		APIURL:   strings.TrimSpace(os.Getenv("VIKUNJA_API_URL")),
		APIToken: strings.TrimSpace(os.Getenv("VIKUNJA_API_TOKEN")),
		Username: strings.TrimSpace(os.Getenv("VIKUNJA_USERNAME")),
		Password: strings.TrimSpace(os.Getenv("VIKUNJA_PASSWORD")),
	}

	if cfg.APIURL == "" {
		return nil, fmt.Errorf(
			"missing VIKUNJA_API_URL. Set it (and VIKUNJA_API_TOKEN or VIKUNJA_USERNAME+VIKUNJA_PASSWORD) in one of:\n  - %s",
			strings.Join(searchPaths(), "\n  - "),
		)
	}
	if cfg.APIToken == "" && (cfg.Username == "" || cfg.Password == "") {
		return nil, fmt.Errorf(
			"missing credentials. Provide VIKUNJA_API_TOKEN or both VIKUNJA_USERNAME and VIKUNJA_PASSWORD in one of:\n  - %s",
			strings.Join(searchPaths(), "\n  - "),
		)
	}
	return cfg, nil
}
