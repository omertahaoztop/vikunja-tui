package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	githubRepo = "omertahaoztop/vikunja-tui"
	assetName  = "vikunja-tui-linux-amd64"
)

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func githubGet(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(raw, out)
}

func download(url string) ([]byte, error) {
	c := &http.Client{Timeout: 5 * time.Minute}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func SelfUpgrade(currentVersion string) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("self-upgrade is only supported on linux/amd64")
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate binary: %w", err)
	}
	binaryPath, _ = filepath.EvalSymlinks(binaryPath)

	fmt.Println("Checking for updates...")
	var rel release
	if err := githubGet("/repos/"+githubRepo+"/releases/latest", &rel); err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if currentVersion != "dev" && currentVersion == rel.TagName {
		fmt.Printf("Already up to date (%s).\n", rel.TagName)
		return nil
	}

	var assetURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			assetURL = a.URL
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no compatible binary found in release %s", rel.TagName)
	}

	if currentVersion == "dev" {
		fmt.Printf("Downloading latest release (%s)...\n", rel.TagName)
	} else {
		fmt.Printf("Updating %s -> %s...\n", currentVersion, rel.TagName)
	}

	data, err := download(assetURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	dir := filepath.Dir(binaryPath)
	tmp, err := os.CreateTemp(dir, ".vikunja-tui-upgrade-")
	if err != nil {
		return fmt.Errorf("permission denied. Try:\n  sudo %s --upgrade", binaryPath)
	}
	tmpPath := tmp.Name()

	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	tmp.Close()

	info, err := os.Stat(binaryPath)
	if err == nil {
		_ = os.Chmod(tmpPath, info.Mode()|0o111)
	} else {
		_ = os.Chmod(tmpPath, 0o755)
	}

	if err := os.Rename(tmpPath, binaryPath); err != nil {
		cleanup()
		return fmt.Errorf("permission denied. Try:\n  sudo %s --upgrade", binaryPath)
	}

	fmt.Printf("Updated to %s.\n", rel.TagName)
	return nil
}
