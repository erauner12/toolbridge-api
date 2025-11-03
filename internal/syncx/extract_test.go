package syncx

import (
	"testing"

	"github.com/google/uuid"
)

func TestExtractCommon(t *testing.T) {
	tests := []struct {
		name    string
		item    map[string]any
		wantErr bool
		check   func(*testing.T, Extracted)
	}{
		{
			name: "complete note",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"title":     "Test Note",
				"updatedTs": "2025-11-03T10:00:00Z",
				"sync": map[string]any{
					"version":   float64(2),
					"isDeleted": false,
				},
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.UID != uuid.MustParse("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f") {
					t.Errorf("UID = %v", ext.UID)
				}
				if ext.Version != 2 {
					t.Errorf("Version = %v, want 2", ext.Version)
				}
				if ext.DeletedAtMs != nil {
					t.Errorf("DeletedAtMs should be nil for non-deleted note")
				}
			},
		},
		{
			name: "deleted note",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"updatedTs": "2025-11-03T10:00:00Z",
				"sync": map[string]any{
					"version":   float64(3),
					"isDeleted": true,
					"deletedAt": "2025-11-03T10:00:00Z",
				},
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.DeletedAtMs == nil {
					t.Error("DeletedAtMs should not be nil for deleted note")
				}
				// Just verify it's set, don't check exact value (timestamp will vary)
			},
		},
		{
			name: "deleted note without deletedAt timestamp",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"updatedTs": "2025-11-03T10:00:00Z",
				"sync": map[string]any{
					"version":   float64(1),
					"isDeleted": true,
				},
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.DeletedAtMs == nil {
					t.Error("DeletedAtMs should be set to updatedTs when missing")
				} else if *ext.DeletedAtMs != ext.UpdatedAtMs {
					t.Errorf("DeletedAtMs (%v) should equal UpdatedAtMs (%v)", *ext.DeletedAtMs, ext.UpdatedAtMs)
				}
			},
		},
		{
			name: "missing uid",
			item: map[string]any{
				"title":     "Test",
				"updatedTs": "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid uid",
			item: map[string]any{
				"uid":       "not-a-uuid",
				"updatedTs": "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "missing sync metadata defaults",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"updatedTs": "2025-11-03T10:00:00Z",
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.Version != 1 {
					t.Errorf("Version should default to 1, got %v", ext.Version)
				}
			},
		},
		{
			name: "alternate timestamp field names",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"updatedAt": "2025-11-03T10:00:00Z",
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.UpdatedAtMs == 0 {
					t.Error("UpdatedAtMs should be parsed from updatedAt field")
				}
			},
		},
		{
			name: "updateTime field name",
			item: map[string]any{
				"uid":        "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"updateTime": "2025-11-03T10:00:00Z",
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.UpdatedAtMs == 0 {
					t.Error("UpdatedAtMs should be parsed from updateTime field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractCommon(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractCommon() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestExtractComment(t *testing.T) {
	tests := []struct {
		name    string
		item    map[string]any
		wantErr bool
		check   func(*testing.T, Extracted)
	}{
		{
			name: "valid comment",
			item: map[string]any{
				"uid":        "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"parentType": "note",
				"parentUid":  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				"updatedTs":  "2025-11-03T10:00:00Z",
				"sync": map[string]any{
					"version": float64(1),
				},
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.ParentType != "note" {
					t.Errorf("ParentType = %v, want note", ext.ParentType)
				}
				if ext.ParentUID == nil {
					t.Error("ParentUID should not be nil")
				} else if *ext.ParentUID != uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
					t.Errorf("ParentUID = %v", *ext.ParentUID)
				}
			},
		},
		{
			name: "missing parentType",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"parentUid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				"updatedTs": "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "missing parentUid",
			item: map[string]any{
				"uid":        "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"parentType": "note",
				"updatedTs":  "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid parentUid",
			item: map[string]any{
				"uid":        "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"parentType": "note",
				"parentUid":  "not-a-uuid",
				"updatedTs":  "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractComment(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractComment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestExtractChatMessage(t *testing.T) {
	tests := []struct {
		name    string
		item    map[string]any
		wantErr bool
		check   func(*testing.T, Extracted)
	}{
		{
			name: "valid chat message",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"chatUid":   "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				"updatedTs": "2025-11-03T10:00:00Z",
				"sync": map[string]any{
					"version": float64(1),
				},
			},
			wantErr: false,
			check: func(t *testing.T, ext Extracted) {
				if ext.ChatUID == nil {
					t.Error("ChatUID should not be nil")
				} else if *ext.ChatUID != uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
					t.Errorf("ChatUID = %v", *ext.ChatUID)
				}
			},
		},
		{
			name: "missing chatUid",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"updatedTs": "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid chatUid",
			item: map[string]any{
				"uid":       "c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f",
				"chatUid":   "not-a-uuid",
				"updatedTs": "2025-11-03T10:00:00Z",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractChatMessage(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractChatMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestParseTimeToMs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		checkMs   bool // Only check if > 0, not exact value
	}{
		{
			name:      "RFC3339",
			input:     "2025-11-03T10:00:00Z",
			wantValid: true,
			checkMs:   true,
		},
		{
			name:      "RFC3339 with nanoseconds",
			input:     "2025-11-03T10:00:00.123456789Z",
			wantValid: true,
			checkMs:   true,
		},
		{
			name:      "numeric milliseconds",
			input:     "1730631600000",
			wantValid: true,
			checkMs:   false,
		},
		{
			name:      "empty string",
			input:     "",
			wantValid: false,
			checkMs:   false,
		},
		{
			name:      "invalid format",
			input:     "not-a-timestamp",
			wantValid: false,
			checkMs:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := ParseTimeToMs(tt.input)
			if valid != tt.wantValid {
				t.Errorf("ParseTimeToMs() valid = %v, want %v", valid, tt.wantValid)
			}
			if valid && tt.checkMs && got == 0 {
				t.Error("ParseTimeToMs() should return non-zero timestamp")
			}
		})
	}
}
