package tools

import (
	"fmt"
	"github.com/google/uuid"
)

type AttachContextParams struct {
	EntityUID  string `json:"entityUid"`
	EntityKind string `json:"entityKind"`
	Title      string `json:"title,omitempty"`
}

func (p *AttachContextParams) Validate() error {
	if _, err := uuid.Parse(p.EntityUID); err != nil {
		return fmt.Errorf("invalid entityUid: %w", err)
	}
	allowedKinds := map[string]bool{"note": true, "task": true, "chat": true}
	if !allowedKinds[p.EntityKind] {
		return fmt.Errorf("invalid entityKind: must be note, task, or chat")
	}
	return nil
}

func (p *AttachContextParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.EntityUID)
}

type DetachContextParams struct {
	EntityUID  string `json:"entityUid"`
	EntityKind string `json:"entityKind"`
}

func (p *DetachContextParams) Validate() error {
	if _, err := uuid.Parse(p.EntityUID); err != nil {
		return fmt.Errorf("invalid entityUid: %w", err)
	}
	allowedKinds := map[string]bool{"note": true, "task": true, "chat": true}
	if !allowedKinds[p.EntityKind] {
		return fmt.Errorf("invalid entityKind: must be note, task, or chat")
	}
	return nil
}

func (p *DetachContextParams) ParseUID() (uuid.UUID, error) {
	return uuid.Parse(p.EntityUID)
}

type ListContextParams struct{}

func (p *ListContextParams) Validate() error {
	return nil
}

type ClearContextParams struct{}

func (p *ClearContextParams) Validate() error {
	return nil
}
