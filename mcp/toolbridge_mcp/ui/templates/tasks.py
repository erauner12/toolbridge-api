"""
HTML templates for Task UI resources.

Converts Task models into HTML for MCP-UI rendering.
NOTE: These are minimal stub templates. A future ticket will add:
- Proper CSS styling
- Interactive elements (complete, edit, delete buttons)
- MCP-UI action handlers (postMessage events)
"""

from typing import Iterable, TYPE_CHECKING
from html import escape

if TYPE_CHECKING:
    from toolbridge_mcp.tools.tasks import Task


def _get_status_icon(status: str) -> str:
    """Get an emoji icon for task status."""
    icons = {
        "todo": "⬜",
        "in_progress": "🔄",
        "done": "✅",
        "archived": "📦",
    }
    return icons.get(status, "⬜")


def _get_priority_class(priority: str) -> str:
    """Get CSS class for priority styling."""
    return f"priority-{priority}" if priority in ("low", "medium", "high") else ""


def render_tasks_list_html(tasks: Iterable["Task"]) -> str:
    """
    Render an HTML list of tasks.

    Args:
        tasks: Iterable of Task objects to display

    Returns:
        HTML string with a styled list of tasks
    """
    tasks_list = list(tasks)

    if not tasks_list:
        return """
        <html>
        <head>
            <style>
                body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 16px; }
                .empty { color: #666; font-style: italic; }
            </style>
        </head>
        <body>
            <h2>✅ Tasks</h2>
            <p class="empty">No tasks found.</p>
        </body>
        </html>
        """

    items_html = ""
    for task in tasks_list:
        title = escape(task.payload.get("title", "Untitled"))
        description = escape(task.payload.get("description", "")[:80])
        if len(task.payload.get("description", "")) > 80:
            description += "..."
        uid = escape(task.uid)
        status = task.payload.get("status", "todo")
        priority = task.payload.get("priority", "")
        due_date = task.payload.get("dueDate", "")

        status_icon = _get_status_icon(status)
        priority_class = _get_priority_class(priority)

        due_html = ""
        if due_date:
            due_html = f'<span class="due-date">📅 {escape(due_date[:10])}</span>'

        priority_html = ""
        if priority:
            priority_html = f'<span class="priority {priority_class}">{escape(priority)}</span>'

        items_html += f"""
        <li class="task-item {priority_class}" data-uid="{uid}" data-status="{escape(status)}">
            <div class="task-header">
                <span class="status-icon">{status_icon}</span>
                <span class="task-title">{title}</span>
                {priority_html}
            </div>
            <div class="task-description">{description}</div>
            <div class="task-meta">
                {due_html}
                <span class="uid">UID: {uid[:8]}...</span>
            </div>
        </li>
        """

    return f"""
    <html>
    <head>
        <style>
            body {{
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                padding: 24px;
                margin: 0;
                background: transparent;
                font-size: 18px;
                color: #e0e0e0;
            }}
            h2 {{
                margin-top: 0;
                color: #ffffff;
                font-size: 32px;
                margin-bottom: 12px;
                text-shadow: 0 2px 4px rgba(0,0,0,0.3);
            }}
            .tasks-list {{ list-style: none; padding: 0; margin: 0; }}
            .task-item {{
                padding: 20px 24px;
                margin-bottom: 16px;
                background: linear-gradient(135deg, #1a4d1a 0%, #2d7a2d 100%);
                border-radius: 16px;
                border-left: 8px solid #4caf50;
                box-shadow: 0 4px 16px rgba(0,0,0,0.3);
            }}
            .task-item.priority-high {{
                border-left-color: #ff5252;
                background: linear-gradient(135deg, #5a1a1a 0%, #8b2d2d 100%);
            }}
            .task-item.priority-medium {{
                border-left-color: #ffc107;
                background: linear-gradient(135deg, #5a4a1a 0%, #8b7a2d 100%);
            }}
            .task-item.priority-low {{
                border-left-color: #90a4ae;
                background: linear-gradient(135deg, #37474f 0%, #546e7a 100%);
            }}
            .task-header {{ display: flex; align-items: center; gap: 14px; margin-bottom: 10px; }}
            .status-icon {{ font-size: 28px; }}
            .task-title {{ font-weight: 700; color: #ffffff; flex: 1; font-size: 22px; }}
            .priority {{
                font-size: 13px;
                padding: 6px 14px;
                border-radius: 8px;
                text-transform: uppercase;
                font-weight: 800;
                letter-spacing: 1px;
            }}
            .priority-high {{ background: #ff5252; color: #ffffff; }}
            .priority-medium {{ background: #ffc107; color: #1a1a1a; }}
            .priority-low {{ background: #78909c; color: #ffffff; }}
            .task-description {{ color: #c8e6c9; font-size: 17px; margin-bottom: 10px; line-height: 1.5; }}
            .task-item.priority-high .task-description {{ color: #ffcdd2; }}
            .task-item.priority-medium .task-description {{ color: #fff8e1; }}
            .task-item.priority-low .task-description {{ color: #cfd8dc; }}
            .task-meta {{ color: #81c784; font-size: 14px; display: flex; gap: 20px; font-weight: 500; }}
            .task-item.priority-high .task-meta {{ color: #ef9a9a; }}
            .task-item.priority-medium .task-meta {{ color: #ffe082; }}
            .task-item.priority-low .task-meta {{ color: #b0bec5; }}
            .due-date {{ color: #4fc3f7; font-weight: 600; }}
            .count {{ color: #a5d6a7; font-size: 18px; margin-bottom: 20px; }}
        </style>
    </head>
    <body>
        <h2>✅ Tasks</h2>
        <p class="count">Showing {len(tasks_list)} task(s)</p>
        <ul class="tasks-list">
            {items_html}
        </ul>
    </body>
    </html>
    """


def render_task_detail_html(task: "Task") -> str:
    """
    Render HTML for a single task detail view.

    Args:
        task: Task object to display

    Returns:
        HTML string with the full task content
    """
    title = escape(task.payload.get("title", "Untitled"))
    description = escape(task.payload.get("description", "No description"))
    uid = escape(task.uid)
    status = task.payload.get("status", "todo")
    priority = task.payload.get("priority", "")
    due_date = task.payload.get("dueDate", "")
    tags = task.payload.get("tags", [])

    status_icon = _get_status_icon(status)
    priority_class = _get_priority_class(priority)

    tags_html = ""
    if tags:
        tags_html = '<div class="tags">' + "".join(
            f'<span class="tag">{escape(str(tag))}</span>' for tag in tags
        ) + "</div>"

    due_html = ""
    if due_date:
        due_html = f'<div class="due-date">📅 Due: {escape(due_date)}</div>'

    priority_html = ""
    if priority:
        priority_html = f'<span class="priority {priority_class}">{escape(priority)}</span>'

    return f"""
    <html>
    <head>
        <style>
            body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 16px; margin: 0; }}
            .task-header {{ display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }}
            h1 {{ margin: 0; color: #333; font-size: 24px; }}
            .status-icon {{ font-size: 28px; }}
            .description {{
                background: #f8f9fa;
                padding: 16px;
                border-radius: 8px;
                white-space: pre-wrap;
                line-height: 1.6;
            }}
            .meta {{ color: #666; font-size: 12px; margin-top: 16px; }}
            .tags {{ margin-top: 12px; }}
            .tag {{
                display: inline-block;
                background: #e9ecef;
                padding: 4px 8px;
                border-radius: 4px;
                font-size: 12px;
                margin-right: 4px;
            }}
            .priority {{
                font-size: 12px;
                padding: 4px 8px;
                border-radius: 4px;
                text-transform: uppercase;
                font-weight: 500;
            }}
            .priority-high {{ background: #f8d7da; color: #721c24; }}
            .priority-medium {{ background: #fff3cd; color: #856404; }}
            .priority-low {{ background: #e2e3e5; color: #383d41; }}
            .due-date {{ color: #007bff; margin-top: 12px; font-weight: 500; }}
            .status {{ margin-top: 8px; color: #666; }}
        </style>
    </head>
    <body>
        <div class="task-header">
            <span class="status-icon">{status_icon}</span>
            <h1>{title}</h1>
            {priority_html}
        </div>
        <div class="status">Status: {escape(status)}</div>
        {due_html}
        {tags_html}
        <h3>Description</h3>
        <div class="description">{description}</div>
        <div class="meta">
            UID: {uid} | Version: {task.version} | Updated: {escape(task.updated_at)}
        </div>
    </body>
    </html>
    """
