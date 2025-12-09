"""Remote DOM templates for Notes UI.

Builds Remote DOM tree structures for native Flutter rendering via RemoteDomView.
"""

from typing import Iterable, TYPE_CHECKING, Dict, Any, List

if TYPE_CHECKING:
    from toolbridge_mcp.tools.notes import Note


def render_notes_list_dom(
    notes: Iterable["Note"],
    limit: int = 20,
    include_deleted: bool = False,
    ui_format: str = "remote-dom",
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for the notes list view.

    Args:
        notes: Iterable of Note models to render
        limit: Current limit value (passed to action buttons)
        include_deleted: Current include_deleted value (passed to action buttons)
        ui_format: Current ui_format value (passed to action buttons)

    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    notes_list = list(notes)

    header_children: List[Dict[str, Any]] = [
        {
            "type": "row",
            "props": {"gap": 8, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": "notes", "size": 24, "color": "primary"}},
                {"type": "text", "props": {"text": "Notes", "style": "headlineMedium"}},
            ],
        },
        {
            "type": "text",
            "props": {
                "text": f"Showing {len(notes_list)} note(s)",
                "style": "bodySmall",
                "color": "onSurface",
            },
        },
    ]

    cards: List[Dict[str, Any]] = []
    for note in notes_list:
        title = (note.payload.get("title") or "Untitled").strip()
        content_raw = (note.payload.get("content") or "").strip()
        preview = content_raw[:100] + ("..." if len(content_raw) > 100 else "")

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
                            "text": preview,
                            "style": "bodySmall",
                            "maxLines": 3,
                            "overflow": "ellipsis",
                        },
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
                                        "toolName": "show_note_ui",
                                        "params": {
                                            "uid": note.uid,
                                            "include_deleted": include_deleted,
                                            "ui_format": ui_format,
                                        },
                                    },
                                },
                            },
                            {
                                "type": "button",
                                "props": {
                                    "label": "Delete",
                                    "variant": "text",
                                    "icon": "delete",
                                },
                                "action": {
                                    "type": "tool",
                                    "payload": {
                                        "toolName": "delete_note_ui",
                                        "params": {
                                            "uid": note.uid,
                                            "limit": limit,
                                            "include_deleted": include_deleted,
                                            "ui_format": ui_format,
                                        },
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
        "children": header_children + cards,
    }


def render_note_detail_dom(
    note: "Note",
    ui_format: str = "remote-dom",
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for the note detail view.

    Args:
        note: Note model to render
        ui_format: Current ui_format value (for consistency)

    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    title = (note.payload.get("title") or "Untitled note").strip()
    content = (note.payload.get("content") or "No content").strip()
    tags = note.payload.get("tags") or []

    tag_nodes: List[Dict[str, Any]] = [
        {
            "type": "container",
            "props": {
                "padding": {"left": 8, "right": 8, "top": 4, "bottom": 4},
                "borderRadius": 16,
            },
            "children": [
                {
                    "type": "text",
                    "props": {"text": str(tag), "style": "labelSmall"},
                }
            ],
        }
        for tag in tags
    ]

    children: List[Dict[str, Any]] = [
        {
            "type": "row",
            "props": {"gap": 8, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": "note", "size": 24, "color": "primary"}},
                {"type": "text", "props": {"text": title, "style": "headlineMedium"}},
            ],
        },
        {
            "type": "text",
            "props": {"text": f"UID: {note.uid}  |  v{note.version}", "style": "caption"},
        },
    ]

    if tag_nodes:
        children.append(
            {
                "type": "row",
                "props": {"gap": 8},
                "children": tag_nodes,
            }
        )

    children.append(
        {
            "type": "card",
            "props": {"padding": 16},
            "children": [
                {
                    "type": "text",
                    "props": {
                        "text": content,
                        "style": "bodyMedium",
                    },
                }
            ],
        }
    )

    return {
        "type": "column",
        "props": {"gap": 12, "padding": 16},
        "children": children,
    }
