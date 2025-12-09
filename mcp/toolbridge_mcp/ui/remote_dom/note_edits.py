"""Remote DOM templates for Note Edit UI.

Builds Remote DOM tree structures for diff preview rendering via RemoteDomView.
Uses design tokens for consistent styling with native ToolBridge UI.
"""

from typing import Dict, Any, List, TYPE_CHECKING

from toolbridge_mcp.ui.remote_dom.design import (
    TextStyle,
    Spacing,
    Layout,
    Color,
    Icon,
    ButtonVariant,
    text_node,
    get_chat_metadata,
)

if TYPE_CHECKING:
    from toolbridge_mcp.tools.notes import Note
    from toolbridge_mcp.utils.diff import DiffHunk


def render_note_edit_diff_dom(
    note: "Note",
    diff_hunks: List["DiffHunk"],
    edit_id: str,
    summary: str | None = None,
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for the note edit diff preview.
    
    Args:
        note: The current note being edited
        diff_hunks: List of diff hunks from compute_line_diff
        edit_id: The edit session ID for action payloads
        summary: Optional summary of the changes
        
    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    title = (note.payload.get("title") or "Untitled note").strip()
    
    children: List[Dict[str, Any]] = [
        # Header with icon and title
        {
            "type": "row",
            "props": {"gap": Spacing.GAP_SM, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": Icon.EDIT, "size": 24, "color": Color.PRIMARY}},
                text_node("Proposed changes", TextStyle.HEADLINE_MEDIUM),
            ],
        },
        # Subtitle with note title and version
        text_node(
            f"{title} (v{note.version})",
            TextStyle.BODY_SMALL,
            Color.ON_SURFACE_VARIANT,
        ),
    ]
    
    # Add summary if provided
    if summary:
        children.append(
            text_node(summary, TextStyle.BODY_MEDIUM, Color.ON_SURFACE),
        )
    
    # Diff card containing all hunks
    diff_children: List[Dict[str, Any]] = []
    for hunk in diff_hunks:
        hunk_node = _render_diff_hunk(hunk)
        if hunk_node:
            diff_children.append(hunk_node)
    
    if diff_children:
        children.append({
            "type": "card",
            "props": {"padding": 20},
            "children": [
                {
                    "type": "column",
                    "props": {
                        "gap": Spacing.GAP_MD,
                        "crossAxisAlignment": "stretch",
                    },
                    "children": diff_children,
                }
            ],
        })
    
    # Action row (Accept / Discard)
    children.append({
        "type": "row",
        "props": {
            "gap": Spacing.GAP_SM,
            "mainAxisAlignment": "end",
        },
        "children": [
            {
                "type": "button",
                "props": {
                    "label": "Discard",
                    "variant": ButtonVariant.TEXT,
                    "icon": Icon.CLOSE,
                },
                "action": {
                    "type": "tool",
                    "payload": {
                        "toolName": "discard_note_edit",
                        "params": {"edit_id": edit_id},
                    },
                },
            },
            {
                "type": "button",
                "props": {
                    "label": "Apply changes",
                    "variant": ButtonVariant.PRIMARY,
                    "icon": Icon.CHECK,
                },
                "action": {
                    "type": "tool",
                    "payload": {
                        "toolName": "apply_note_edit",
                        "params": {"edit_id": edit_id},
                    },
                },
            },
        ],
    })
    
    # Build root props
    root_props = {
        "gap": Spacing.SECTION_GAP,
        "padding": 24,
        "fullWidth": True,
        "crossAxisAlignment": "stretch",
    }
    if Layout.MAX_WIDTH_DETAIL is not None:
        root_props["maxWidth"] = Layout.MAX_WIDTH_DETAIL
    
    return {
        "type": "column",
        "props": root_props,
        "children": children,
    }


# GitHub-style diff colors (hex)
# These work well in both light and dark modes
DIFF_ADDED_BG = "#1c4428"      # Dark green background
DIFF_ADDED_TEXT = "#3fb950"    # Bright green text
DIFF_REMOVED_BG = "#5c1a1b"    # Dark red background
DIFF_REMOVED_TEXT = "#f85149"  # Bright red text
DIFF_CONTEXT_BG = "#21262d"    # Dark gray background
DIFF_CONTEXT_TEXT = "#8b949e"  # Gray text


def _render_diff_line(text: str, is_added: bool) -> Dict[str, Any]:
    """Render a single diff line with +/- prefix (GitHub-style)."""
    prefix = "+ " if is_added else "- "
    text_color = DIFF_ADDED_TEXT if is_added else DIFF_REMOVED_TEXT
    bg_color = DIFF_ADDED_BG if is_added else DIFF_REMOVED_BG

    return {
        "type": "container",
        "props": {
            "padding": 8,
            "color": bg_color,
        },
        "children": [
            {
                "type": "text",
                "props": {
                    "text": f"{prefix}{text}",
                    "style": TextStyle.BODY_SMALL,
                    "color": text_color,
                },
            },
        ],
    }


def _render_context_line(text: str) -> Dict[str, Any]:
    """Render an unchanged context line."""
    return {
        "type": "container",
        "props": {
            "padding": 8,
            "color": DIFF_CONTEXT_BG,
        },
        "children": [
            {
                "type": "text",
                "props": {
                    "text": f"  {text}",
                    "style": TextStyle.BODY_SMALL,
                    "color": DIFF_CONTEXT_TEXT,
                },
            },
        ],
    }


def _render_diff_hunk(hunk: "DiffHunk") -> Dict[str, Any] | None:
    """Render a single diff hunk as GitHub-style diff lines."""
    if hunk.kind == "unchanged":
        if not hunk.original:
            return None
        # Show context lines (up to 3 lines for brevity)
        lines = hunk.original.split('\n')
        if len(lines) > 3:
            # Show first line, ellipsis, last line
            children = [
                _render_context_line(lines[0]),
                {
                    "type": "container",
                    "props": {"padding": 4, "color": DIFF_CONTEXT_BG},
                    "children": [
                        text_node(f"  ... ({len(lines) - 2} more lines)", TextStyle.BODY_SMALL, DIFF_CONTEXT_TEXT),
                    ],
                },
                _render_context_line(lines[-1]),
            ]
        else:
            children = [_render_context_line(line) for line in lines]

        return {
            "type": "column",
            "props": {"gap": 0, "crossAxisAlignment": "stretch"},
            "children": children,
        }

    elif hunk.kind == "removed":
        lines = hunk.original.split('\n')
        return {
            "type": "column",
            "props": {"gap": 0, "crossAxisAlignment": "stretch"},
            "children": [_render_diff_line(line, is_added=False) for line in lines],
        }

    elif hunk.kind == "added":
        lines = hunk.proposed.split('\n')
        return {
            "type": "column",
            "props": {"gap": 0, "crossAxisAlignment": "stretch"},
            "children": [_render_diff_line(line, is_added=True) for line in lines],
        }

    elif hunk.kind == "modified":
        children: List[Dict[str, Any]] = []

        # Show removed lines first (red with -)
        if hunk.original:
            for line in hunk.original.split('\n'):
                children.append(_render_diff_line(line, is_added=False))

        # Then added lines (green with +)
        if hunk.proposed:
            for line in hunk.proposed.split('\n'):
                children.append(_render_diff_line(line, is_added=True))

        return {
            "type": "column",
            "props": {"gap": 0, "crossAxisAlignment": "stretch"},
            "children": children,
        }

    return None


def render_note_edit_success_dom(
    note: "Note",
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for successful note edit confirmation.

    Shows the updated note content after applying changes.

    Args:
        note: The updated note after applying changes

    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    title = (note.payload.get("title") or "Untitled note").strip()
    content = (note.payload.get("content") or "").strip()
    tags = note.payload.get("tags") or []

    children: List[Dict[str, Any]] = [
        # Success header
        {
            "type": "row",
            "props": {"gap": Spacing.GAP_SM, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": Icon.CHECK_CIRCLE, "size": 24, "color": Color.PRIMARY}},
                text_node("Changes applied", TextStyle.HEADLINE_MEDIUM),
            ],
        },
        # Subtitle
        text_node(
            f"Updated to v{note.version}",
            TextStyle.BODY_SMALL,
            Color.ON_SURFACE_VARIANT,
        ),
    ]

    # Updated content card
    content_children: List[Dict[str, Any]] = [
        # Note title
        text_node(title, TextStyle.TITLE_LARGE, Color.ON_SURFACE),
    ]

    # Tags row if present
    if tags:
        tag_children = [
            {
                "type": "chip",
                "props": {"label": tag},
            }
            for tag in tags[:5]  # Limit to 5 tags
        ]
        content_children.append({
            "type": "row",
            "props": {"gap": Spacing.GAP_XS},
            "children": tag_children,
        })

    # Note content
    if content:
        content_children.append({
            "type": "divider",
            "props": {"margin": Spacing.GAP_SM},
        })
        content_children.append(
            text_node(content, TextStyle.BODY_MEDIUM, Color.ON_SURFACE),
        )

    children.append({
        "type": "card",
        "props": {"padding": 20},
        "children": [
            {
                "type": "column",
                "props": {"gap": Spacing.GAP_MD, "crossAxisAlignment": "stretch"},
                "children": content_children,
            }
        ],
    })

    # Build root props
    root_props = {
        "gap": Spacing.SECTION_GAP,
        "padding": 24,
        "fullWidth": True,
        "crossAxisAlignment": "stretch",
    }
    if Layout.MAX_WIDTH_DETAIL is not None:
        root_props["maxWidth"] = Layout.MAX_WIDTH_DETAIL

    return {
        "type": "column",
        "props": root_props,
        "children": children,
    }


def render_note_edit_discarded_dom(
    title: str,
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for discarded note edit confirmation.
    
    Args:
        title: The note title
        
    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    children: List[Dict[str, Any]] = [
        # Info header
        {
            "type": "row",
            "props": {"gap": Spacing.GAP_SM, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": Icon.CLOSE, "size": 24, "color": Color.ON_SURFACE_VARIANT}},
                text_node("Changes discarded", TextStyle.HEADLINE_MEDIUM),
            ],
        },
        text_node(
            f"Pending edits for '{title}' have been discarded.",
            TextStyle.BODY_MEDIUM,
            Color.ON_SURFACE_VARIANT,
        ),
    ]
    
    # Build root props
    root_props = {
        "gap": Spacing.GAP_MD,
        "padding": 24,
        "fullWidth": True,
        "crossAxisAlignment": "stretch",
    }
    
    return {
        "type": "column",
        "props": root_props,
        "children": children,
    }


def render_note_edit_error_dom(
    error_message: str,
    note_uid: str | None = None,
) -> Dict[str, Any]:
    """
    Build Remote DOM tree for note edit error.
    
    Args:
        error_message: The error message to display
        note_uid: Optional note UID for retry suggestion
        
    Returns:
        Root node dict compatible with RemoteDomNode.fromJson
    """
    children: List[Dict[str, Any]] = [
        # Error header
        {
            "type": "row",
            "props": {"gap": Spacing.GAP_SM, "crossAxisAlignment": "center"},
            "children": [
                {"type": "icon", "props": {"icon": Icon.ERROR, "size": 24, "color": Color.ERROR}},
                text_node("Failed to apply changes", TextStyle.HEADLINE_MEDIUM),
            ],
        },
        text_node(error_message, TextStyle.BODY_MEDIUM, Color.ON_ERROR_CONTAINER),
    ]
    
    if note_uid:
        children.append(
            text_node(
                "The note may have been modified. Please re-run edit_note_ui to create a fresh diff.",
                TextStyle.BODY_SMALL,
                Color.ON_SURFACE_VARIANT,
            )
        )
    
    # Build root props
    root_props = {
        "gap": Spacing.GAP_MD,
        "padding": 24,
        "fullWidth": True,
        "crossAxisAlignment": "stretch",
        "color": Color.ERROR_CONTAINER,
        "borderRadius": 12,
    }
    
    return {
        "type": "column",
        "props": root_props,
        "children": children,
    }
