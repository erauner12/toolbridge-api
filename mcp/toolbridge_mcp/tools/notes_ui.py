"""
MCP-UI tools for Note display.

Provides UI-enhanced versions of note tools that return both text fallback
and interactive HTML/Remote DOM for MCP-UI compatible hosts.
"""

from typing import Annotated, List, Union

from pydantic import Field
from loguru import logger
from mcp.types import TextContent, EmbeddedResource

from toolbridge_mcp.mcp_instance import mcp
from toolbridge_mcp.tools.notes import list_notes, get_note, delete_note, Note, NotesListResponse
from toolbridge_mcp.ui.resources import build_ui_with_text_and_dom, UIContent, UIFormat
from toolbridge_mcp.ui.templates import notes as notes_templates
from toolbridge_mcp.ui.remote_dom import notes as notes_dom_templates


@mcp.tool()
async def list_notes_ui(
    limit: Annotated[int, Field(ge=1, le=100, description="Max notes to display")] = 20,
    include_deleted: Annotated[bool, Field(description="Include deleted notes")] = False,
    ui_format: Annotated[
        str,
        Field(
            description="UI format: 'html' (default), 'remote-dom', or 'both'",
            pattern="^(html|remote-dom|both)$",
        ),
    ] = "html",
) -> List[Union[TextContent, EmbeddedResource]]:
    """
    Display notes with interactive UI (MCP-UI).

    This tool returns both a text summary (for non-UI hosts) and an interactive
    HTML or Remote DOM view (for MCP-UI compatible hosts like Goose, Nanobot, or LibreChat).

    The UI view shows a styled list of notes with:
    - Note titles and content previews
    - Metadata (UID, version)
    - Visual styling for easy scanning

    Args:
        limit: Maximum number of notes to display (1-100, default 20)
        include_deleted: Whether to include soft-deleted notes (default False)
        ui_format: UI format to return - 'html' (default), 'remote-dom', or 'both'

    Returns:
        List containing TextContent (summary) and UIResource(s) (HTML and/or Remote DOM)

    Examples:
        # Show recent notes with HTML UI (default)
        >>> await list_notes_ui(limit=10)

        # Include deleted notes with Remote DOM UI
        >>> await list_notes_ui(include_deleted=True, ui_format="remote-dom")

        # Return both HTML and Remote DOM
        >>> await list_notes_ui(ui_format="both")
    """
    logger.info(f"Rendering notes UI: limit={limit}, include_deleted={include_deleted}, ui_format={ui_format}")

    # Reuse existing data tool to fetch notes
    notes_response: NotesListResponse = await list_notes(
        limit=limit,
        cursor=None,
        include_deleted=include_deleted,
    )

    html: str | None = None
    remote_dom: dict | None = None

    # Only render HTML when needed (html or both)
    if ui_format in ("html", "both"):
        html = notes_templates.render_notes_list_html(
            notes_response.items,
            limit=limit,
            include_deleted=include_deleted,
        )

    # Only render Remote DOM when needed (remote-dom or both)
    if ui_format in ("remote-dom", "both"):
        remote_dom = notes_dom_templates.render_notes_list_dom(
            notes_response.items,
            limit=limit,
            include_deleted=include_deleted,
            ui_format=ui_format,
        )

    # Human-readable summary (shown even if host ignores UIResource)
    count = len(notes_response.items)
    summary = f"Displaying {count} note(s) (limit={limit}, include_deleted={include_deleted})"

    if notes_response.next_cursor:
        summary += f"\nMore notes available (cursor: {notes_response.next_cursor[:20]}...)"

    ui_uri = "ui://toolbridge/notes/list"

    return build_ui_with_text_and_dom(
        uri=ui_uri,
        html=html,
        remote_dom=remote_dom,
        text_summary=summary,
        ui_format=UIFormat(ui_format),
    )


@mcp.tool()
async def show_note_ui(
    uid: Annotated[str, Field(description="UID of the note to display")],
    include_deleted: Annotated[bool, Field(description="Allow deleted notes")] = False,
    ui_format: Annotated[
        str,
        Field(
            description="UI format: 'html' (default), 'remote-dom', or 'both'",
            pattern="^(html|remote-dom|both)$",
        ),
    ] = "html",
) -> List[Union[TextContent, EmbeddedResource]]:
    """
    Display a single note with interactive UI (MCP-UI).

    Shows a detailed view of a note including:
    - Full title and content
    - Tags and status badges
    - Version and timestamp metadata

    Args:
        uid: Unique identifier of the note (UUID format)
        include_deleted: Whether to allow viewing soft-deleted notes (default False)
        ui_format: UI format to return - 'html' (default), 'remote-dom', or 'both'

    Returns:
        List containing TextContent (summary) and UIResource(s) (HTML and/or Remote DOM detail view)

    Examples:
        # Show a specific note with HTML UI
        >>> await show_note_ui("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f")

        # Show a deleted note with Remote DOM UI
        >>> await show_note_ui("c1d9b7dc-...", include_deleted=True, ui_format="remote-dom")
    """
    logger.info(f"Rendering note UI: uid={uid}, include_deleted={include_deleted}, ui_format={ui_format}")

    # Fetch the note using existing data tool
    note: Note = await get_note(uid=uid, include_deleted=include_deleted)

    html: str | None = None
    remote_dom: dict | None = None

    # Only render HTML when needed (html or both)
    if ui_format in ("html", "both"):
        html = notes_templates.render_note_detail_html(note)

    # Only render Remote DOM when needed (remote-dom or both)
    if ui_format in ("remote-dom", "both"):
        remote_dom = notes_dom_templates.render_note_detail_dom(note, ui_format=ui_format)

    # Human-readable summary (guard against null values)
    title = note.payload.get("title") or "Untitled note"
    content = note.payload.get("content") or ""
    content_preview = content[:100]
    if len(content) > 100:
        content_preview += "..."

    summary = f"Note: {title}\n\n{content_preview}\n\n(UID: {uid}, version: {note.version})"

    ui_uri = f"ui://toolbridge/notes/{uid}"

    return build_ui_with_text_and_dom(
        uri=ui_uri,
        html=html,
        remote_dom=remote_dom,
        text_summary=summary,
        ui_format=UIFormat(ui_format),
    )


@mcp.tool()
async def delete_note_ui(
    uid: Annotated[str, Field(description="UID of the note to delete")],
    limit: Annotated[int, Field(ge=1, le=100, description="Max notes to display in refreshed list")] = 20,
    include_deleted: Annotated[bool, Field(description="Include deleted notes in refreshed list")] = False,
    ui_format: Annotated[
        str,
        Field(
            description="UI format: 'html' (default), 'remote-dom', or 'both'",
            pattern="^(html|remote-dom|both)$",
        ),
    ] = "html",
) -> List[Union[TextContent, EmbeddedResource]]:
    """
    Delete a note and return updated UI (MCP-UI).

    Soft deletes the note and returns an updated notes list with interactive HTML or Remote DOM.

    Args:
        uid: Unique identifier of the note to delete
        limit: Maximum notes to display in refreshed list (preserves list context)
        include_deleted: Whether to include deleted notes (preserves list context)
        ui_format: UI format to return - 'html' (default), 'remote-dom', or 'both'

    Returns:
        List containing TextContent (summary) and UIResource(s) (updated HTML and/or Remote DOM list)

    Examples:
        >>> await delete_note_ui("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f")

        # Delete with custom list context and Remote DOM
        >>> await delete_note_ui("c1d9b7dc-...", limit=50, include_deleted=True, ui_format="remote-dom")
    """
    logger.info(f"Deleting note UI: uid={uid}, limit={limit}, include_deleted={include_deleted}, ui_format={ui_format}")

    # Perform the delete using the underlying tool
    deleted_note: Note = await delete_note(uid=uid)
    note_title = deleted_note.payload.get("title", "Note")

    # Fetch updated notes list with preserved context
    notes_response: NotesListResponse = await list_notes(limit=limit, include_deleted=include_deleted)

    html: str | None = None
    remote_dom: dict | None = None

    # Only render HTML when needed (html or both)
    if ui_format in ("html", "both"):
        html = notes_templates.render_notes_list_html(
            notes_response.items,
            limit=limit,
            include_deleted=include_deleted,
        )

    # Only render Remote DOM when needed (remote-dom or both)
    if ui_format in ("remote-dom", "both"):
        remote_dom = notes_dom_templates.render_notes_list_dom(
            notes_response.items,
            limit=limit,
            include_deleted=include_deleted,
            ui_format=ui_format,
        )

    summary = f"Deleted '{note_title}' - {len(notes_response.items)} note(s) remaining"

    return build_ui_with_text_and_dom(
        uri="ui://toolbridge/notes/list",
        html=html,
        remote_dom=remote_dom,
        text_summary=summary,
        ui_format=UIFormat(ui_format),
    )
