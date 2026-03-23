import json
import os
import platform
import stat
import sys
import tempfile
from urllib.error import HTTPError
from urllib.request import Request, urlopen

GITHUB_REPO = "omertahaoztop/vikunja-tui"
ASSET_NAME = "vikunja-tui-linux-amd64"


def _github_get(path: str) -> dict:
    url = f"https://api.github.com{path}"
    req = Request(url, headers={"Accept": "application/vnd.github.v3+json"})
    with urlopen(req) as r:
        result = json.loads(r.read())
    if not isinstance(result, dict):
        raise RuntimeError(f"Unexpected response from {path}")
    return result


def _get_current_binary() -> str | None:
    if getattr(sys, "frozen", False):
        return os.path.realpath(sys.executable)
    return None


def self_upgrade(current_version: str) -> None:
    binary_path = _get_current_binary()
    if not binary_path:
        print("Not running as a compiled binary. Use git pull to update from source.")
        sys.exit(1)

    if platform.system() != "Linux" or platform.machine() not in ("x86_64", "AMD64"):
        print("Self-upgrade is only supported on Linux amd64.")
        sys.exit(1)

    print("Checking for updates...")

    try:
        release = _github_get(f"/repos/{GITHUB_REPO}/releases/latest")
    except HTTPError as e:
        print(f"Failed to check for updates: HTTP {e.code}")
        sys.exit(1)
    except Exception as e:
        print(f"Failed to check for updates: {e}")
        sys.exit(1)

    tag = release["tag_name"]

    if current_version != "dev" and current_version == tag:
        print(f"Already up to date ({tag}).")
        sys.exit(0)

    asset_url = None
    for asset in release.get("assets", []):
        if asset["name"] == ASSET_NAME:
            asset_url = asset["browser_download_url"]
            break

    if not asset_url:
        print(f"No compatible binary found in release {tag}.")
        sys.exit(1)

    if current_version == "dev":
        print(f"Downloading latest release ({tag})...")
    else:
        print(f"Updating {current_version} -> {tag}...")

    try:
        req = Request(asset_url)
        with urlopen(req) as r:
            data = r.read()
    except Exception as e:
        print(f"Download failed: {e}")
        sys.exit(1)

    binary_dir = os.path.dirname(binary_path)
    tmp_path: str | None = None
    try:
        fd, tmp_path = tempfile.mkstemp(dir=binary_dir, prefix=".vikunja-tui-upgrade-")
        os.write(fd, data)
        os.close(fd)
        original_mode = os.stat(binary_path).st_mode
        os.chmod(tmp_path, original_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
        os.replace(tmp_path, binary_path)
    except PermissionError:
        if tmp_path:
            try:
                os.unlink(tmp_path)
            except OSError:
                pass
        print(f"Permission denied. Try:\n  sudo {binary_path} --upgrade")
        sys.exit(1)
    except Exception:
        if tmp_path:
            try:
                os.unlink(tmp_path)
            except OSError:
                pass
        raise

    print(f"Updated to {tag}.")
