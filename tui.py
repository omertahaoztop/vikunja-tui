from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Container, Horizontal, VerticalScroll, Vertical
from textual.widgets import (
    Header,
    Footer,
    Button,
    Label,
    Tree,
    Input,
    LoadingIndicator,
    Static,
)
from textual.screen import ModalScreen, Screen
from textual import work
from textual.widget import Widget
from rich.text import Text

from client import VikunjaClient


def _one_line(text: str, maxlen: int = 0) -> str:
    """Collapse multiline text to a single line (no hard truncation; Rich handles overflow)."""
    if not text:
        return "Untitled"
    line = text.replace("\r", "").split("\n")[0].strip()
    if not line:
        return "Untitled"
    # Only hard-truncate when an explicit maxlen is requested (e.g. notify messages)
    if maxlen and len(line) > maxlen:
        line = line[: maxlen - 1] + "\u2026"
    return line


# --------------------------------------------------------------------------
# Dashboard
# --------------------------------------------------------------------------


class ProjectBoardTree(Screen):
    """Dashboard -- pick a project."""

    def compose(self) -> ComposeResult:
        yield Header()
        with Vertical(id="dashboard_container"):
            yield Static("vikunja \u00b7 tui", classes="dashboard_logo")
            yield Tree("Projects", id="project_tree")
        yield Footer()

    def on_mount(self) -> None:
        self._load()

    @work(thread=True)
    def _load(self) -> None:
        try:
            vikunja = VikunjaClient.get_instance()
            projects = vikunja.projects
            self.app.call_from_thread(self._populate, projects)
        except Exception as e:
            self.app.call_from_thread(
                self.notify, f"Connection error: {e}", severity="error"
            )

    def _populate(self, projects) -> None:
        tree = self.query_one("#project_tree", Tree)
        tree.root.expand()

        # Group projects by parent_project_id for nested display
        children_map: dict[int, list] = {}
        roots = []
        for p in projects:
            pid = p.parent_project_id
            if pid and pid > 0:
                children_map.setdefault(pid, []).append(p)
            else:
                roots.append(p)

        def _add_children(node, parent_id):
            for child in children_map.get(parent_id, []):
                child_children = children_map.get(child._data["id"], [])
                if child_children:
                    cnode = node.add(child.title, expand=True)
                    cnode.data = child
                    _add_children(cnode, child._data["id"])
                else:
                    node.add_leaf(child.title, data=child)

        for project in roots:
            children = children_map.get(project._data["id"], [])
            if children:
                pnode = tree.root.add(project.title, expand=True)
                pnode.data = project
                _add_children(pnode, project._data["id"])
            else:
                tree.root.add_leaf(project.title, data=project)

    def on_tree_node_selected(self, event: Tree.NodeSelected) -> None:
        if event.node.data:
            self.app.push_screen(BoardScreen(event.node.data))


# --------------------------------------------------------------------------
# Task widget -- a focusable Label row
# --------------------------------------------------------------------------


class TaskWidget(Widget):
    """Single-line focusable task row."""

    can_focus = True
    DEFAULT_CSS = "TaskWidget { height: 1; min-height: 1; max-height: 1; }"

    def __init__(self, vtask, **kwargs):
        super().__init__(**kwargs)
        self.vtask = vtask

    def on_focus(self) -> None:
        self.refresh()

    def on_blur(self) -> None:
        self.refresh()

    def render(self) -> Text:
        title = _one_line(self.vtask.title)
        bullet = "\u25b6 " if self.has_focus else "\u00b7 "
        avail = max(4, (self.size.width or 38) - 4)
        if self.vtask.done:
            t = Text(bullet + title, no_wrap=True, overflow="ellipsis", style="strike dim")
        else:
            t = Text(bullet + title, no_wrap=True, overflow="ellipsis")
        t.truncate(avail, overflow="ellipsis")
        return t


# --------------------------------------------------------------------------
# Bucket column
# --------------------------------------------------------------------------


class BucketColumn(VerticalScroll):
    """Vertical column for one Vikunja bucket."""

    can_focus = True

    BINDINGS = [
        ("down", "next_task", "\u2193"),
        ("up", "prev_task", "\u2191"),
    ]

    def __init__(self, bucket, **kwargs):
        super().__init__(**kwargs)
        self.bucket = bucket

    def _tasks(self) -> list[TaskWidget]:
        return list(self.query(TaskWidget))

    def action_next_task(self) -> None:
        self._move(1)

    def action_prev_task(self) -> None:
        self._move(-1)

    def _move(self, d: int) -> None:
        tasks = self._tasks()
        if not tasks:
            return
        focused = self.screen.focused
        if focused == self:
            tasks[0].focus()
            return
        if focused in tasks:
            i = tasks.index(focused) + d
            if 0 <= i < len(tasks):
                tasks[i].focus()

    def on_focus(self) -> None:
        tasks = self._tasks()
        if tasks:
            tasks[0].focus()

    def compose(self) -> ComposeResult:
        tasks = list(self.bucket.tasks)
        title = self.bucket.title or "\u2014"
        yield Label(f"{title}  [{len(tasks)}]", classes="list_header")
        for task in tasks:
            yield TaskWidget(task, classes="card")

    def refresh_header(self) -> None:
        count = len(self._tasks())
        self.query_one(".list_header", Label).update(f"{self.bucket.title}  [{count}]")


# --------------------------------------------------------------------------
# Modals
# --------------------------------------------------------------------------


class InputModal(ModalScreen[str | None]):
    BINDINGS = [("escape", "cancel", "Cancel")]

    def __init__(self, prompt: str):
        super().__init__()
        self.prompt_text = prompt

    def compose(self) -> ComposeResult:
        with Container(classes="modal_box"):
            yield Label(self.prompt_text, classes="modal_title")
            yield Input(id="modal_input", classes="modal_input")
            with Horizontal(classes="modal_row"):
                yield Button("OK", variant="primary", id="ok")
                yield Button("Cancel", id="cancel")

    def on_button_pressed(self, e: Button.Pressed) -> None:
        if e.button.id == "ok":
            self.dismiss(self.query_one(Input).value or None)
        else:
            self.dismiss(None)

    def on_input_submitted(self, e: Input.Submitted) -> None:
        self.dismiss(e.value or None)

    def action_cancel(self) -> None:
        self.dismiss(None)


class ConfirmModal(ModalScreen[bool]):
    BINDINGS = [("escape", "no", "No"), ("y", "yes", "Yes")]

    def __init__(self, prompt: str):
        super().__init__()
        self.prompt_text = prompt

    def compose(self) -> ComposeResult:
        with Container(classes="modal_box"):
            yield Label(self.prompt_text, classes="modal_title")
            with Horizontal(classes="modal_row"):
                yield Button("Yes  [y]", variant="error", id="yes")
                yield Button("No [esc]", id="no")

    def on_button_pressed(self, e: Button.Pressed) -> None:
        self.dismiss(e.button.id == "yes")

    def action_yes(self) -> None:
        self.dismiss(True)

    def action_no(self) -> None:
        self.dismiss(False)


class DetailModal(ModalScreen):
    BINDINGS = [("escape", "close", "Close"), ("q", "close", "Close")]

    def __init__(self, title: str, body: str):
        super().__init__()
        self._title = title
        self._body = body

    def compose(self) -> ComposeResult:
        with Container(classes="modal_box detail_box"):
            yield Label(self._title, classes="modal_title")
            yield Label(self._body or "No description.", classes="modal_body")
            yield Button("Close  [esc]", variant="primary", id="close")

    def on_button_pressed(self, _: Button.Pressed) -> None:
        self.dismiss()

    def action_close(self) -> None:
        self.dismiss()


# --------------------------------------------------------------------------
# Board screen
# --------------------------------------------------------------------------


class BoardScreen(Screen):
    """Kanban board for a Vikunja project."""

    BINDINGS = [
        ("escape", "app.pop_screen", "\u2190 Back"),
        ("a", "add_task", "+ Task"),
        ("d", "delete_task", "Delete"),
        ("D", "clear_bucket", "Clear Bucket"),
        ("c", "mark_done", "\u2713 Toggle Done"),
        ("enter", "view_details", "Details"),
        ("r", "reload", "Reload"),
    ]

    def __init__(self, project, **kwargs):
        super().__init__(**kwargs)
        self._project = project
        self._buckets: list = []

    def compose(self) -> ComposeResult:
        yield Header()
        yield LoadingIndicator(id="loading")
        yield Horizontal(id="board")
        yield Footer()

    def on_mount(self) -> None:
        self._fetch()

    @work(thread=True)
    def _fetch(self) -> None:
        try:
            buckets = VikunjaClient.get_instance().load_project_board(self._project)
            self.app.call_from_thread(self._show_board, buckets)
        except Exception as e:
            self.app.call_from_thread(
                self.notify, f"Failed to load project: {e}", severity="error"
            )

    def _show_board(self, buckets) -> None:
        self._buckets = buckets
        self.query_one("#loading").display = False
        container = self.query_one("#board", Horizontal)
        container.remove_children()
        for bucket in buckets:
            if bucket.title:
                container.mount(BucketColumn(bucket, classes="list_col"))
        self.sub_title = self._project.title

    def action_reload(self) -> None:
        self.query_one("#loading").display = True
        self.query_one("#board", Horizontal).remove_children()
        self._fetch()

    # -- navigation -----------------------------------------------------------

    def _shift_col(self, d: int) -> None:
        cols = list(self.query(BucketColumn))
        if not cols:
            return
        cur = self._focused_col()
        idx = (cols.index(cur) + d) % len(cols) if cur in cols else 0
        cols[idx].focus()

    def on_key(self, e) -> None:
        if e.key == "tab":
            self._shift_col(1)
            e.stop()
        elif e.key == "shift+tab":
            self._shift_col(-1)
            e.stop()
        elif e.key == "right":
            self._shift_col(1)
            e.stop()
        elif e.key == "left":
            self._shift_col(-1)
            e.stop()

    def _focused_task(self) -> TaskWidget | None:
        f = self.app.focused
        return f if isinstance(f, TaskWidget) else None

    def _focused_col(self) -> BucketColumn | None:
        f = self.app.focused
        if isinstance(f, TaskWidget) and isinstance(f.parent, BucketColumn):
            return f.parent
        if isinstance(f, BucketColumn):
            return f
        return None

    # -- actions --------------------------------------------------------------

    def action_add_task(self) -> None:
        col = self._focused_col()
        if col is None:
            try:
                col = self.query_one(BucketColumn)
            except Exception:
                self.notify("No bucket found.", severity="warning")
                return

        def _done(title: str | None) -> None:
            if not title:
                return
            try:
                vikunja = VikunjaClient.get_instance()
                task = vikunja.create_task(self._project, col.bucket, title)
                col.mount(TaskWidget(task, classes="card"))
                col.refresh_header()
                self.notify(f"Added: {_one_line(title, 30)}")
            except Exception as ex:
                self.notify(f"Error: {ex}", severity="error")

        self.app.push_screen(InputModal("New task title:"), _done)

    def action_delete_task(self) -> None:
        tw = self._focused_task()
        if tw is None:
            self.notify("No task selected.", severity="warning")
            return

        def _done(ok: bool) -> None:
            if not ok:
                return
            try:
                col = tw.parent
                tw.vtask.delete()
                tw.remove()
                if isinstance(col, BucketColumn):
                    col.refresh_header()
                self.notify("Deleted.")
            except Exception as ex:
                self.notify(f"Error: {ex}", severity="error")

        self.app.push_screen(
            ConfirmModal(f"Delete '{_one_line(tw.vtask.title, 40)}'?"), _done
        )

    def action_mark_done(self) -> None:
        tw = self._focused_task()
        if tw is None:
            self.notify("No task selected.", severity="warning")
            return

        try:
            new_state = not tw.vtask.done
            tw.vtask.update(done=new_state)
            tw.refresh()
            if new_state:
                self.notify("Marked as done.")
            else:
                self.notify("Marked as not done.")
        except Exception as ex:
            self.notify(f"Error: {ex}", severity="error")

    def action_view_details(self) -> None:
        tw = self._focused_task()
        if tw:
            self.app.push_screen(DetailModal(tw.vtask.title, tw.vtask.description))

    def action_clear_bucket(self) -> None:
        col = self._focused_col()
        if col is None:
            self.notify("No bucket selected.", severity="warning")
            return
        tasks = col._tasks()
        if not tasks:
            self.notify("Bucket is already empty.", severity="warning")
            return

        def _done(ok: bool) -> None:
            if not ok:
                return
            deleted, failed = 0, 0
            for tw in tasks:
                try:
                    tw.vtask.delete()
                    tw.remove()
                    deleted += 1
                except Exception:
                    failed += 1
            col.refresh_header()
            sev = "warning" if failed else "information"
            self.notify(
                f"{deleted} task(s) deleted."
                + (f" {failed} failed." if failed else ""),
                severity=sev,
            )

        title = col.bucket.title or "this bucket"
        self.app.push_screen(
            ConfirmModal(f"Delete all {len(tasks)} tasks in '{title}'?"), _done
        )
