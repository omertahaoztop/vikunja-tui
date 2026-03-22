"""Vikunja API client -- direct HTTP, no external API dependency."""

import os
import json
from pathlib import Path
from typing import Optional
from urllib.request import Request, urlopen
from urllib.error import HTTPError
from dotenv import load_dotenv

CONFIG_SEARCH_PATHS = [
    Path("/etc/default/vikunja-tui"),
    Path.home() / ".config" / "vikunja-tui" / "config",
    Path.cwd() / ".env",
]

# ---------------------------------------------------------------------------
# Lightweight model objects (plain dicts wrapped in a class for attr access)
# ---------------------------------------------------------------------------


class _Obj:
    """Wrap a raw API dict so attributes are accessible as obj.field."""

    def __init__(self, data: dict, routes: "Routes"):
        self._data = data
        self._routes = routes

    def __getattr__(self, name):
        try:
            return self._data[name]
        except KeyError:
            raise AttributeError(name)

    def __repr__(self):
        return f"{self.__class__.__name__}({self._data.get('title', self._data.get('id', '?'))})"


class Task(_Obj):
    @property
    def title(self) -> str:
        return self._data.get("title") or "Untitled"

    @property
    def description(self) -> str:
        return self._data.get("description") or ""

    @property
    def done(self) -> bool:
        return self._data.get("done", False)

    @property
    def bucket_id(self) -> int:
        return self._data.get("bucket_id", 0)

    def delete(self):
        self._routes.delete(f"/tasks/{self._data['id']}")

    def update(self, **fields):
        merged = {**self._data, **fields}
        result = self._routes.post(f"/tasks/{self._data['id']}", merged)
        self._data.update(result)
        return self

    def mark_done(self):
        return self.update(done=True)


class Bucket(_Obj):
    def __init__(self, data: dict, tasks: list, routes: "Routes"):
        super().__init__(data, routes)
        self._tasks = tasks

    @property
    def title(self) -> Optional[str]:
        return self._data.get("title")

    @property
    def tasks(self) -> list:
        return self._tasks

    @property
    def is_done_bucket(self) -> bool:
        return self._data.get("done", False)


class Project(_Obj):
    @property
    def title(self) -> str:
        return self._data.get("title", "Untitled")

    @property
    def description(self) -> str:
        return self._data.get("description") or ""

    @property
    def parent_project_id(self) -> int:
        return self._data.get("parent_project_id", 0)

    @property
    def views(self) -> list:
        return self._data.get("views") or []


# ---------------------------------------------------------------------------
# HTTP layer
# ---------------------------------------------------------------------------


class Routes:
    def __init__(self, base_url: str, token: str):
        self._base = base_url.rstrip("/")
        self._token = token

    def _request(self, method: str, path: str, data: Optional[dict] = None) -> dict:
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self._token}",
        }
        body = json.dumps(data).encode() if data is not None else None
        req = Request(self._base + path, data=body, headers=headers, method=method)
        try:
            with urlopen(req) as r:
                raw = r.read()
                return json.loads(raw) if raw else {}
        except HTTPError as e:
            msg = e.read().decode(errors="replace")
            raise RuntimeError(f"Vikunja API {method} {path} -> {e.code}: {msg}") from e

    def get(self, path: str) -> dict | list:
        return self._request("GET", path)

    def post(self, path: str, data: dict) -> dict:
        return self._request("POST", path, data)

    def put(self, path: str, data: dict) -> dict:
        return self._request("PUT", path, data)

    def delete(self, path: str) -> dict:
        return self._request("DELETE", path)


# ---------------------------------------------------------------------------
# High-level Vikunja client
# ---------------------------------------------------------------------------


def _ensure_api_v1(url: str) -> str:
    """Normalise base URL to end with /api/v1."""
    clean = url.rstrip("/")
    if clean.endswith("/api/v1"):
        return clean
    if clean.endswith("/api"):
        return clean + "/v1"
    return clean + "/api/v1"


class VikunjaAPI:
    """Authenticated Vikunja client."""

    def __init__(self, base_url: str, token: str):
        """Initialize with base URL and pre-existing API/JWT token."""
        self._routes = Routes(_ensure_api_v1(base_url), token)
        self._me_name: Optional[str] = None

    @classmethod
    def from_credentials(
        cls, base_url: str, username: str, password: str
    ) -> "VikunjaAPI":
        """Create client by logging in with username/password."""
        login_url = _ensure_api_v1(base_url) + "/login"
        req = Request(
            login_url,
            data=json.dumps({"username": username, "password": password}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urlopen(req) as r:
                resp = json.loads(r.read())
        except HTTPError as e:
            raise RuntimeError(f"Login failed ({e.code}): {e.read().decode()}") from e

        token = resp.get("token")
        if not token:
            raise RuntimeError(f"Login failed: no token in response. Got: {resp}")

        return cls(base_url, token)

    @property
    def me_name(self) -> str:
        if self._me_name is None:
            data = self._routes.get("/user")
            if isinstance(data, dict):
                self._me_name = data.get("name") or data.get("username") or "Unknown"
            else:
                self._me_name = "Unknown"
        return self._me_name

    @property
    def projects(self) -> list:
        """Return list of Project objects."""
        resp = self._routes.get("/projects")
        if not isinstance(resp, list):
            resp = resp.get("items", resp.get("data", []))
        return [Project(p, self._routes) for p in resp if not p.get("is_archived")]

    def _find_kanban_view(self, project: Project) -> Optional[dict]:
        """Find the kanban view for a project.

        Checks views embedded in the project response first; falls back to
        fetching /projects/{id}/views if needed.
        """
        views = project.views
        if not views:
            fetched = self._routes.get(f"/projects/{project._data['id']}/views")
            views = fetched if isinstance(fetched, list) else []

        # Prefer kanban view, fall back to first view
        for v in views:
            vk = v.get("view_kind")
            if vk == "kanban" or vk == 3:
                return v
        return views[0] if views else None

    def load_project_board(self, project: Project) -> list:
        """Load kanban buckets + tasks for a project. Returns list of Bucket objects."""
        project_id = project._data["id"]
        view = self._find_kanban_view(project)
        if not view:
            return []

        view_id = view["id"]

        # Store view metadata on project for later use (task creation, done bucket)
        project._data["_view_id"] = view_id
        project._data["_done_bucket_id"] = view.get("done_bucket_id", 0)
        project._data["_default_bucket_id"] = view.get("default_bucket_id", 0)

        # Fetch tasks for this view.
        # The Vikunja API returns an **array of bucket objects**, each
        # carrying a nested ``tasks`` list (may be ``null``).
        raw_resp = self._routes.get(f"/projects/{project_id}/views/{view_id}/tasks")

        items: list = raw_resp if isinstance(raw_resp, list) else []

        # Detect whether items look like buckets-with-nested-tasks
        # (each item has "tasks" key) or plain task objects.
        is_bucket_list = items and isinstance(items[0], dict) and "tasks" in items[0]

        if is_bucket_list:
            # Each item is a bucket dict with a nested "tasks" array.
            buckets = []
            for b in items:
                raw_tasks = b.get("tasks") or []
                task_objs = [
                    Task(t, self._routes) for t in raw_tasks if not t.get("done")
                ]
                buckets.append(Bucket(b, task_objs, self._routes))
            return buckets

        # Fallback: items are plain task dicts — group by bucket_id and
        # fetch the bucket list separately.
        tasks_by_bucket: dict[int, list] = {}
        for t in items:
            if isinstance(t, dict):
                bid = t.get("bucket_id", 0)
                tasks_by_bucket.setdefault(bid, []).append(t)

        raw_buckets = self._routes.get(
            f"/projects/{project_id}/views/{view_id}/buckets"
        )
        if not isinstance(raw_buckets, list):
            raw_buckets = []

        buckets = []
        for b in raw_buckets:
            bid = b["id"]
            raw_tasks = tasks_by_bucket.get(bid, [])
            task_objs = [Task(t, self._routes) for t in raw_tasks if not t.get("done")]
            buckets.append(Bucket(b, task_objs, self._routes))

        return buckets

        # Shape B / C -- need explicit bucket list
        raw_buckets = self._routes.get(
            f"/projects/{project_id}/views/{view_id}/buckets"
        )
        if not isinstance(raw_buckets, list):
            raw_buckets = []

        buckets = []
        for b in raw_buckets:
            bid = b["id"]
            raw_tasks = tasks_by_bucket.get(bid, [])
            task_objs = [Task(t, self._routes) for t in raw_tasks if not t.get("done")]
            buckets.append(Bucket(b, task_objs, self._routes))

        return buckets

    def create_task(self, project: Project, bucket: Bucket, title: str) -> Task:
        """Create a new task in a project, assigned to a specific bucket."""
        project_id = project._data["id"]
        view_id = project._data.get("_view_id")

        task_data: dict = {"title": title, "bucket_id": bucket._data["id"]}
        result = self._routes.put(f"/projects/{project_id}/tasks", task_data)

        # Vikunja may ignore bucket_id on creation; move explicitly via
        # the view-level task-bucket endpoint.
        task_id = result.get("id")
        if task_id and view_id:
            try:
                self._routes.post(
                    f"/projects/{project_id}/views/{view_id}/buckets/{bucket._data['id']}",
                    {"task_id": task_id},
                )
            except Exception:
                pass  # bucket_id in PUT body may have already worked

        return Task(result, self._routes)


# ---------------------------------------------------------------------------
# Singleton accessor
# ---------------------------------------------------------------------------


class VikunjaClient:
    _instance: Optional[VikunjaAPI] = None

    @classmethod
    def _load_config(cls) -> None:
        for config_path in CONFIG_SEARCH_PATHS:
            if config_path.exists():
                load_dotenv(config_path)
                return
        load_dotenv()

    @classmethod
    def get_instance(cls) -> VikunjaAPI:
        if cls._instance is None:
            cls._load_config()
            url = os.getenv("VIKUNJA_API_URL")
            token = os.getenv("VIKUNJA_API_TOKEN")
            username = os.getenv("VIKUNJA_USERNAME")
            password = os.getenv("VIKUNJA_PASSWORD")

            if not url:
                search_paths = "\n  - ".join(str(p) for p in CONFIG_SEARCH_PATHS)
                raise ValueError(
                    "Missing Vikunja config. "
                    f"Set VIKUNJA_API_URL (and either VIKUNJA_API_TOKEN or VIKUNJA_USERNAME+VIKUNJA_PASSWORD) in:\n  - {search_paths}\n"
                    "Or export them as environment variables."
                )

            if token:
                cls._instance = VikunjaAPI(url, token)
            elif username and password:
                cls._instance = VikunjaAPI.from_credentials(url, username, password)
            else:
                search_paths = "\n  - ".join(str(p) for p in CONFIG_SEARCH_PATHS)
                raise ValueError(
                    "Missing Vikunja credentials. "
                    "Provide VIKUNJA_API_TOKEN or both VIKUNJA_USERNAME and VIKUNJA_PASSWORD in:\n"
                    f"  - {search_paths}\n"
                    "Or export them as environment variables."
                )

        return cls._instance


if __name__ == "__main__":
    try:
        client = VikunjaClient.get_instance()
        print(f"Connected as: {client.me_name}")
        for project in client.projects:
            print(f"  Project: {project.title}")
            buckets = client.load_project_board(project)
            for bucket in buckets:
                print(f"    Bucket: {bucket.title} ({len(bucket.tasks)} tasks)")
    except Exception as e:
        print(f"Connection failed: {e}")
