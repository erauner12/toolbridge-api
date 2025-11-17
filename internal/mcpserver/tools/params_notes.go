package tools

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type ListNotesParams struct {
	Cursor         *string `json:"cursor,omitempty"`
	Limit          *int    `json:"limit,omitempty"`
	IncludeDeleted bool    `json:"includeDeleted,omitempty"`
}

func (p *ListNotesParams) Validate() error {
	if p.Limit != nil && (*p.Limit < 1 || *p.Limit > 1000) {
		return fmt.Errorf("limit must be between 1 and 1000")
	}
	return nil
}

type GetNoteParams struct {
	UID            string `json:"uid"`
	IncludeDeleted bool   `json:"includeDeleted,omitempty"`
}

func (p *GetNoteParams) Validate() error {
	if _, err := uuid.Parse(p.UID); err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}
	return nil
}

func (p *GetNoteParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.UID)
}

type CreateNoteParams struct {
	UID     *string        `json:"uid,omitempty"`
	Payload map[string]any `json:"payload"`
}

func (p *CreateNoteParams) Validate() error {
	if p.Payload == nil || len(p.Payload) == 0 {
		return fmt.Errorf("payload is required and cannot be empty")
	}
	if p.UID != nil {
		if _, err := uuid.Parse(*p.UID); err != nil {
			return fmt.Errorf("invalid uid: %w", err)
		}
	}
	return nil
}

func (p *CreateNoteParams) ParseUID() (*uuid.UUID, error) {
	if p.UID == nil {
		return nil, nil
	}
	uid, err := uuid.Parse(*p.UID)
	if err != nil {
		return nil, err
	}
	return &uid, nil
}

type UpdateNoteParams struct {
	UID     string         `json:"uid"`
	Payload map[string]any `json:"payload"`
	Version *int           `json:"version,omitempty"`
}

func (p *UpdateNoteParams) Validate() error {
	if _, err := uuid.Parse(p.UID); err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}
	if p.Payload == nil || len(p.Payload) == 0 {
		return fmt.Errorf("payload is required and cannot be empty")
	}
	return nil
}

func (p *UpdateNoteParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.UID)
}

type PatchNoteParams struct {
	UID     string         `json:"uid"`
	Partial map[string]any `json:"partial"`
}

func (p *PatchNoteParams) Validate() error {
	if _, err := uuid.Parse(p.UID); err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}
	if p.Partial == nil || len(p.Partial) == 0 {
		return fmt.Errorf("partial is required and cannot be empty")
	}
	return nil
}

func (p *PatchNoteParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.UID)
}

type DeleteNoteParams struct {
	UID string `json:"uid"`
}

func (p *DeleteNoteParams) Validate() error {
	if _, err := uuid.Parse(p.UID); err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}
	return nil
}

func (p *DeleteNoteParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.UID)
}

type ArchiveNoteParams struct {
	UID string `json:"uid"`
}

func (p *ArchiveNoteParams) Validate() error {
	if _, err := uuid.Parse(p.UID); err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}
	return nil
}

func (p *ArchiveNoteParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.UID)
}

type ProcessNoteParams struct {
	UID      string         `json:"uid"`
	Action   string         `json:"action"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (p *ProcessNoteParams) Validate() error {
	if _, err := uuid.Parse(p.UID); err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}
	allowedActions := map[string]bool{
		"pin":    true,
		"unpin":  true,
		"tag":    true,
		"untag":  true,
		"export": true,
	}
	if !allowedActions[p.Action] {
		return fmt.Errorf("invalid action: %s", p.Action)
	}
	return nil
}

func (p *ProcessNoteParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.UID)
}

func DecodeNotesParams[T any](raw json.RawMessage, validator interface{ Validate() error }) (T, error) {
	var params T
	if err := json.Unmarshal(raw, &params); err != nil {
		return params, NewToolError(ErrCodeInvalidParams, "Invalid parameters: "+err.Error(), nil)
	}
	if err := validator.Validate(); err != nil {
		return params, NewToolError(ErrCodeInvalidParams, err.Error(), nil)
	}
	return params, nil
}
