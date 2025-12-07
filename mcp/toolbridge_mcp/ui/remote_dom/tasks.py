"""Remote DOM templates for Tasks UI.

Builds Remote DOM tree structures for native Flutter rendering via RemoteDomView.
"""

from typing import Iterable, TYPE_CHECKING, Dict, Any, List

if TYPE_CHECKING:
    from toolbridge_mcp.tools.tasks import Task


def render_tasks_list_dom(
    tasks: Iterable["Task"],
    limit: int = 20,
    include_deleted: bool = False,
    ui_format: str = "remote-dom",
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for the tasks list view.

    Args:
        tasks: Iterable of Task models to render
        limit: Current limit value (passed to action buttons)
        include_deleted: Current include_deleted value (passed to action buttons)
        ui_format: Current ui_format value (passed to action buttons)

    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    tasks_list = list(tasks)

    header: List[Dict[str, Any]] = [
        {
            "type": "row",
            "props": {"gap": 8, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": "task", "size": 24, "color": "primary"}},
                {"type": "text", "props": {"text": "Tasks", "style": "headlineMedium"}},
            ],
        },
        {
            "type": "text",
            "props": {
                "text": f"Showing {len(tasks_list)} task(s)",
                "style": "bodySmall",
                "color": "onSurface",
            },
        },
    ]

    cards: List[Dict[str, Any]] = []
    for task in tasks_list:
        title = (task.payload.get("title") or "Untitled task").strip()
        desc_raw = (task.payload.get("description") or "").strip()
        description = desc_raw[:80] + ("..." if len(desc_raw) > 80 else "")
        status = task.payload.get("status") or "todo"
        priority = task.payload.get("priority") or ""
        due_date = task.payload.get("dueDate") or ""

        status_label = status.replace("_", " ").title()

        meta_text_parts = [f"Status: {status_label}"]
        if priority:
            meta_text_parts.append(f"Priority: {priority}")
        if due_date:
            meta_text_parts.append(f"Due: {due_date[:10]}")
        meta_text = " | ".join(meta_text_parts)

        # Choose button label based on status
        if status == "done":
            primary_button = {
                "label": "Archive",
                "toolName": "archive_task_ui",
                "params": {
                    "uid": task.uid,
                    "limit": limit,
                    "include_deleted": include_deleted,
                    "ui_format": ui_format,
                },
                "icon": "archive",
            }
        else:
            primary_button = {
                "label": "Complete",
                "toolName": "process_task_ui",
                "params": {
                    "uid": task.uid,
                    "action": "complete",
                    "limit": limit,
                    "include_deleted": include_deleted,
                    "ui_format": ui_format,
                },
                "icon": "task_alt",
            }

        cards.append(
            {
                "type": "card",
                "props": {"padding": 16},
                "children": [
                    {
                        "type": "text",
                        "props": {"text": title, "style": "titleMedium"},
                    },
                    {
                        "type": "text",
                        "props": {
                            "text": description,
                            "style": "bodySmall",
                            "maxLines": 3,
                            "overflow": "ellipsis",
                        },
                    },
                    {
                        "type": "text",
                        "props": {"text": meta_text, "style": "caption"},
                    },
                    {
                        "type": "row",
                        "props": {"gap": 8},
                        "children": [
                            {
                                "type": "button",
                                "props": {
                                    "label": "View",
                                    "variant": "text",
                                    "icon": "visibility",
                                },
                                "action": {
                                    "type": "tool",
                                    "payload": {
                                        "toolName": "show_task_ui",
                                        "params": {
                                            "uid": task.uid,
                                            "include_deleted": include_deleted,
                                            "ui_format": ui_format,
                                        },
                                    },
                                },
                            },
                            {
                                "type": "button",
                                "props": {
                                    "label": primary_button["label"],
                                    "variant": "text",
                                    "icon": primary_button["icon"],
                                },
                                "action": {
                                    "type": "tool",
                                    "payload": {
                                        "toolName": primary_button["toolName"],
                                        "params": primary_button["params"],
                                    },
                                },
                            },
                        ],
                    },
                ],
            }
        )

    return {
        "type": "column",
        "props": {"gap": 12, "padding": 16},
        "children": header + cards,
    }


def render_task_detail_dom(
    task: "Task",
    ui_format: str = "remote-dom",
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for the task detail view.

    Args:
        task: Task model to render
        ui_format: Current ui_format value (for consistency)

    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    title = (task.payload.get("title") or "Untitled task").strip()
    description = (task.payload.get("description") or "No description").strip()
    status = task.payload.get("status") or "todo"
    priority = task.payload.get("priority") or ""
    due_date = task.payload.get("dueDate") or ""
    tags = task.payload.get("tags") or []

    status_label = status.replace("_", " ").title()

    tag_nodes: List[Dict[str, Any]] = [
        {
            "type": "container",
            "props": {
                "padding": {"left": 8, "right": 8, "top": 4, "bottom": 4},
                "borderRadius": 16,
            },
            "children": [
                {"type": "text", "props": {"text": str(tag), "style": "labelSmall"}},
            ],
        }
        for tag in tags
    ]

    children: List[Dict[str, Any]] = [
        {
            "type": "row",
            "props": {"gap": 8, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": "task", "size": 24, "color": "primary"}},
                {"type": "text", "props": {"text": title, "style": "headlineMedium"}},
            ],
        },
        {
            "type": "text",
            "props": {"text": f"Status: {status_label}", "style": "bodySmall"},
        },
    ]

    if priority:
        children.append(
            {
                "type": "text",
                "props": {"text": f"Priority: {priority}", "style": "bodySmall"},
            }
        )

    if due_date:
        children.append(
            {
                "type": "text",
                "props": {"text": f"Due: {due_date}", "style": "bodySmall"},
            }
        )

    if tag_nodes:
        children.append(
            {"type": "row", "props": {"gap": 8}, "children": tag_nodes}
        )

    children.append(
        {
            "type": "card",
            "props": {"padding": 16},
            "children": [
                {
                    "type": "text",
                    "props": {"text": description, "style": "bodyMedium"},
                }
            ],
        }
    )

    children.append(
        {
            "type": "text",
            "props": {
                "text": f"UID: {task.uid}  |  v{task.version}",
                "style": "caption",
            },
        }
    )

    return {
        "type": "column",
        "props": {"gap": 12, "padding": 16},
        "children": children,
    }
