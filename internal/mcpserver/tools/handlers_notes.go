package tools

import (
	"context"
	"encoding/json"

	"github.com/erauner12/toolbridge-api/internal/mcpserver/client"
)

// Notes tool handlers

func HandleListNotes(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params ListNotesParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
		}
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	opts := client.ListOpts{
		Limit:          100, // default
		IncludeDeleted: params.IncludeDeleted,
	}
	if params.Cursor != nil {
		opts.Cursor = *params.Cursor
	}
	if params.Limit != nil {
		opts.Limit = *params.Limit
	}

	result, err := notesClient.List(ctx, opts)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return result, nil
}

func HandleGetNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params GetNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	uid, err := params.ParseUID()
	if err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	item, err := notesClient.Get(ctx, uid, params.IncludeDeleted)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return item, nil
}

func HandleCreateNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params CreateNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	// If client provided UID, add it to payload
	if params.UID != nil {
		uid, err := params.ParseUID()
		if err != nil {
			return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
		}
		if uid != nil {
			params.Payload["uid"] = uid.String()
		}
	}

	item, err := notesClient.Create(ctx, params.Payload)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return item, nil
}

func HandleUpdateNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params UpdateNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	uid, err := params.ParseUID()
	if err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	item, err := notesClient.Update(ctx, uid, params.Payload, params.Version)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return item, nil
}

func HandlePatchNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params PatchNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	uid, err := params.ParseUID()
	if err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	item, err := notesClient.Patch(ctx, uid, params.Partial)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return item, nil
}

func HandleDeleteNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params DeleteNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	uid, err := params.ParseUID()
	if err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	if err := notesClient.Delete(ctx, uid); err != nil {
		return nil, WrapClientError(err)
	}

	return map[string]any{"success": true}, nil
}

func HandleArchiveNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params ArchiveNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	uid, err := params.ParseUID()
	if err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	item, err := notesClient.Archive(ctx, uid)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return item, nil
}

func HandleProcessNote(ctx context.Context, tc *ToolContext, raw json.RawMessage) (interface{}, error) {
	var params ProcessNoteParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := params.Validate(); err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}

	uid, err := params.ParseUID()
	if err != nil {
		return nil, NewToolError(ErrCodeInvalidParams, "Invalid UID: "+err.Error(), nil)
	}

	notesClient, err := tc.GetClient("notes")
	if err != nil {
		return nil, err
	}

	item, err := notesClient.Process(ctx, uid, params.Action, params.Metadata)
	if err != nil {
		return nil, WrapClientError(err)
	}

	return item, nil
}
