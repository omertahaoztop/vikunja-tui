# Vikunja TUI

A Bubble Tea-based Terminal User Interface (TUI) for [Vikunja](https://vikunja.io), the open-source to-do app.

## Features

-   **Dashboard**: View and select from your available projects (with nested project support).
-   **Kanban View**: Interact with your project buckets and tasks.
-   **Task Management**:
    -   Create new tasks.
    -   Edit task title and description.
    -   View task details.
    -   Mark tasks as done / not done.
    -   Move tasks between buckets.
    -   Delete tasks.
    -   Clear entire buckets.
-   **Search / Filter**: Live in-board task filtering (`/`).
-   **Task Badges**: Priority, due date, and label chips on each card.
-   **Themes**: Cycle through GitHub Dark, Dracula, Nord, and Gruvbox (`t`).
-   **Help Overlay**: Full keybinding reference on demand (`?`).
-   **Mouse Support**: Cell-motion mouse tracking.
-   **Auto-sync**: Board refreshes every 5 seconds.
-   **Keyboard Navigation**: Full keyboard support with vim-style bindings.
-   **Self-upgrade**: Update to latest release with `--upgrade`.

## Installation

### Release binary (Linux)

1.  Download the latest `vikunja-tui-linux-amd64` from Releases:
    https://github.com/omertahaoztop/vikunja-tui/releases

2.  Make it executable:
    ```bash
    chmod +x vikunja-tui-linux-amd64
    ```

3.  Run:
    ```bash
    ./vikunja-tui-linux-amd64
    ```

Example (system-wide install with `wget`):

```bash
# Download the binary and make it executable
sudo wget https://github.com/omertahaoztop/vikunja-tui/releases/latest/download/vikunja-tui-linux-amd64 -O /usr/local/bin/vikunja-tui
sudo chmod +x /usr/local/bin/vikunja-tui
# Run
vikunja-tui
```

### From source

1.  Clone the repository:
    ```bash
    git clone https://github.com/omertahaoztop/vikunja-tui.git
    cd vikunja-tui
    ```

2.  Build the binary:
    ```bash
    go build -o vikunja-tui .
    ./vikunja-tui
    ```

## Configuration

Configuration is loaded from the first file found (in order):
1. `/etc/default/vikunja-tui` (system-wide, recommended for binary installs)
2. `~/.config/vikunja-tui/config` (user-specific)
3. `.env` in current directory (for development)

### API Token (recommended)

Create an API token in Vikunja: Settings > API Tokens.

```bash
sudo tee /etc/default/vikunja-tui << 'EOF'
VIKUNJA_API_URL=https://your-vikunja-instance.com
VIKUNJA_API_TOKEN=your_api_token
EOF
sudo chmod 600 /etc/default/vikunja-tui
```

### Username/Password (self-hosted only)

```bash
sudo tee /etc/default/vikunja-tui << 'EOF'
VIKUNJA_API_URL=https://your-vikunja-instance.com
VIKUNJA_USERNAME=your_username
VIKUNJA_PASSWORD=your_password
EOF
sudo chmod 600 /etc/default/vikunja-tui
```

### From source

```bash
cp .env.example .env
# Edit .env with your credentials
```

### Required variables

```
VIKUNJA_API_URL=https://your-vikunja-instance.com
```

Plus one of:
- `VIKUNJA_API_TOKEN` (recommended)
- `VIKUNJA_USERNAME` + `VIKUNJA_PASSWORD` (self-hosted only)

## Usage

Run the application:

### Binary

```bash
vikunja-tui
```

### From source

```bash
go build -o vikunja-tui . && ./vikunja-tui
```

### Updating

```bash
vikunja-tui --upgrade
```

If installed system-wide:

```bash
sudo vikunja-tui --upgrade
```

### Key Bindings

| Key | Action |
| :--- | :--- |
| `Tab` / `Right` | Next Bucket |
| `Shift+Tab` / `Left` | Previous Bucket |
| `Down` | Next Task |
| `Up` | Previous Task |
| `a` | Add Task |
| `e` | Edit Task |
| `d` | Delete Task |
| `D` (Shift+D) | Clear Bucket (delete all tasks) |
| `c` | Toggle Done |
| `<` / `>` | Move Task to Prev / Next Bucket |
| `/` | Search / Filter |
| `t` | Cycle Theme |
| `?` | Help Overlay |
| `Enter` | View Task Details |
| `r` | Reload Board |
| `Esc` | Back / Cancel |
| `q` | Quit |

## Tech Stack

- **[Go](https://go.dev/)** — Compiled, statically-linked language
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — Terminal UI framework (Elm architecture)
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — TUI components (text input, text area)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** — Style and layout
## Disclaimer

This project is created for **personal and educational purposes only**. It is not affiliated with, endorsed by, or directly supported by the official Vikunja project. Use at your own risk.
