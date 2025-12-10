/**
 * Tool definitions with ChatGPT Apps SDK metadata.
 *
 * These definitions wrap the Python MCP tools and add:
 * - openai/outputTemplate pointing to HTML widgets
 * - openai/widgetAccessible for interactive widgets
 * - Status messages for tool invocation UI
 */

export interface ToolDefinition {
  name: string;
  description: string;
  inputSchema: {
    type: "object";
    properties: Record<string, unknown>;
    required?: string[];
  };
  // ChatGPT Apps metadata
  outputTemplate: string;
  invokingMessage?: string;
  invokedMessage?: string;
  widgetAccessible?: boolean;
}

export const TOOL_DEFINITIONS: ToolDefinition[] = [
  // ════════════════════════════════════════════════════════════════
  // NOTES UI TOOLS
  // ════════════════════════════════════════════════════════════════
  {
    name: "list_notes_ui",
    description:
      "Display notes in an interactive UI widget. Shows a styled card list with View and Delete buttons.",
    inputSchema: {
      type: "object",
      properties: {
        limit: {
          type: "integer",
          description: "Maximum number of notes to display (1-100)",
          default: 20,
        },
        include_deleted: {
          type: "boolean",
          description: "Whether to include soft-deleted notes",
          default: false,
        },
        ui_format: {
          type: "string",
          description: "UI format: 'html' (default), 'remote-dom', or 'both'",
          default: "html",
        },
      },
    },
    outputTemplate: "ui://toolbridge/notes/list",
    invokingMessage: "Loading your notes...",
    invokedMessage: "Notes ready",
    widgetAccessible: true,
  },
  {
    name: "show_note_ui",
    description:
      "Display a single note in a detailed UI view with full content, tags, and metadata.",
    inputSchema: {
      type: "object",
      properties: {
        uid: {
          type: "string",
          description: "Unique identifier of the note (UUID format)",
        },
        include_deleted: {
          type: "boolean",
          description: "Whether to allow viewing soft-deleted notes",
          default: false,
        },
        ui_format: {
          type: "string",
          description: "UI format: 'html' (default), 'remote-dom', or 'both'",
          default: "html",
        },
      },
      required: ["uid"],
    },
    outputTemplate: "ui://toolbridge/notes/detail",
    invokingMessage: "Loading note...",
    invokedMessage: "Note ready",
    widgetAccessible: true,
  },
  {
    name: "delete_note_ui",
    description:
      "Delete a note and return an updated notes list UI. Performs soft delete.",
    inputSchema: {
      type: "object",
      properties: {
        uid: {
          type: "string",
          description: "Unique identifier of the note to delete",
        },
        limit: {
          type: "integer",
          description: "Maximum notes to display in refreshed list",
          default: 20,
        },
        include_deleted: {
          type: "boolean",
          description: "Include deleted notes in refreshed list",
          default: false,
        },
        ui_format: {
          type: "string",
          description: "UI format: 'html' (default), 'remote-dom', or 'both'",
          default: "html",
        },
      },
      required: ["uid"],
    },
    outputTemplate: "ui://toolbridge/notes/list",
    invokingMessage: "Deleting note...",
    invokedMessage: "Note deleted",
    widgetAccessible: true,
  },
  {
    name: "edit_note_ui",
    description:
      "Propose changes to a note and display a diff preview UI with Accept/Discard buttons.",
    inputSchema: {
      type: "object",
      properties: {
        uid: {
          type: "string",
          description: "Unique identifier of the note to edit",
        },
        new_content: {
          type: "string",
          description: "Proposed full rewritten note content (markdown)",
        },
        summary: {
          type: "string",
          description: "Short human summary of the change (optional)",
        },
        ui_format: {
          type: "string",
          description: "UI format: 'html' (default), 'remote-dom', or 'both'",
          default: "html",
        },
      },
      required: ["uid", "new_content"],
    },
    outputTemplate: "ui://toolbridge/notes/edit",
    invokingMessage: "Preparing edit preview...",
    invokedMessage: "Edit preview ready",
    widgetAccessible: true,
  },

  // ════════════════════════════════════════════════════════════════
  // TASKS UI TOOLS
  // ════════════════════════════════════════════════════════════════
  {
    name: "list_tasks_ui",
    description:
      "Display tasks in an interactive UI widget. Shows a styled list with status and actions.",
    inputSchema: {
      type: "object",
      properties: {
        limit: {
          type: "integer",
          description: "Maximum number of tasks to display (1-100)",
          default: 20,
        },
        include_deleted: {
          type: "boolean",
          description: "Whether to include soft-deleted tasks",
          default: false,
        },
        ui_format: {
          type: "string",
          description: "UI format: 'html' (default), 'remote-dom', or 'both'",
          default: "html",
        },
      },
    },
    outputTemplate: "ui://toolbridge/tasks/list",
    invokingMessage: "Loading your tasks...",
    invokedMessage: "Tasks ready",
    widgetAccessible: true,
  },

  // ════════════════════════════════════════════════════════════════
  // NOTE EDIT ACTION TOOLS (called by widgets)
  // ════════════════════════════════════════════════════════════════
  {
    name: "apply_note_edit",
    description:
      "Apply a pending note edit session. Called when user clicks Accept on diff preview.",
    inputSchema: {
      type: "object",
      properties: {
        edit_id: {
          type: "string",
          description: "ID of the pending note edit session",
        },
        ui_format: {
          type: "string",
          default: "html",
        },
      },
      required: ["edit_id"],
    },
    outputTemplate: "ui://toolbridge/notes/detail",
    invokingMessage: "Applying changes...",
    invokedMessage: "Changes applied",
    widgetAccessible: true,
  },
  {
    name: "discard_note_edit",
    description:
      "Discard a pending note edit session. Called when user clicks Discard on diff preview.",
    inputSchema: {
      type: "object",
      properties: {
        edit_id: {
          type: "string",
          description: "ID of the pending note edit session",
        },
        ui_format: {
          type: "string",
          default: "html",
        },
      },
      required: ["edit_id"],
    },
    outputTemplate: "ui://toolbridge/notes/list",
    invokingMessage: "Discarding changes...",
    invokedMessage: "Changes discarded",
    widgetAccessible: true,
  },
  {
    name: "accept_note_edit_hunk",
    description: "Accept a specific diff hunk in a pending edit session.",
    inputSchema: {
      type: "object",
      properties: {
        edit_id: {
          type: "string",
          description: "ID of the pending note edit session",
        },
        hunk_id: {
          type: "string",
          description: "ID of the diff hunk to accept (e.g., 'h1', 'h2')",
        },
        ui_format: {
          type: "string",
          default: "html",
        },
      },
      required: ["edit_id", "hunk_id"],
    },
    outputTemplate: "ui://toolbridge/notes/edit",
    invokingMessage: "Accepting change...",
    invokedMessage: "Change accepted",
    widgetAccessible: true,
  },
  {
    name: "reject_note_edit_hunk",
    description: "Reject a specific diff hunk in a pending edit session.",
    inputSchema: {
      type: "object",
      properties: {
        edit_id: {
          type: "string",
          description: "ID of the pending note edit session",
        },
        hunk_id: {
          type: "string",
          description: "ID of the diff hunk to reject (e.g., 'h1', 'h2')",
        },
        ui_format: {
          type: "string",
          default: "html",
        },
      },
      required: ["edit_id", "hunk_id"],
    },
    outputTemplate: "ui://toolbridge/notes/edit",
    invokingMessage: "Rejecting change...",
    invokedMessage: "Change rejected",
    widgetAccessible: true,
  },
];
