# Vikunja TUI

A Textual-based Terminal User Interface (TUI) for [Vikunja](https://vikunja.io), the open-source to-do app.

## Features

-   **Dashboard**: View and select from your available projects.
-   **Kanban View**: Interact with your project buckets and tasks.
-   **Task Management**:
    -   Create new tasks.
    -   View task details.
    -   Mark tasks as done.
    -   Delete tasks.
-   **Keyboard Navigation**: Full keyboard support for navigating projects, buckets, and tasks.

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

2.  Create and activate a virtual environment:
    ```bash
    python -m venv .venv
    source .venv/bin/activate  # On Windows: .venv\Scripts\activate
    ```

3.  Install dependencies:
    ```bash
    pip install textual python-dotenv
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
python main.py
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
| `d` | Delete Task |
| `D` (Shift+D) | Clear Bucket (delete all tasks) |
| `c` | Mark Task as Done |
| `Enter` | View Task Details |
| `Esc` | Back / Cancel |

## Disclaimer

This project is created for **personal and educational purposes only**. It is not affiliated with, endorsed by, or directly supported by the official Vikunja project. Use at your own risk.
