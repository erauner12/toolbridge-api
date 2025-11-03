package syncx

import (
	"testing"

	"github.com/google/uuid"
)

func TestEncodeCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   Cursor
		expected string
	}{
		{
			name: "normal cursor",
			cursor: Cursor{
				Ms:  1730635200000,
				UID: uuid.MustParse("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f"),
			},
			expected: "MTczMDYzNTIwMDAwMHxjMWQ5YjdkYy1hMWIyLTRjM2QtOWU4Zi03YTZiNWM0ZDNlMmY",
		},
		{
			name: "zero timestamp",
			cursor: Cursor{
				Ms:  0,
				UID: uuid.MustParse("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f"),
			},
			expected: "MHxjMWQ5YjdkYy1hMWIyLTRjM2QtOWU4Zi03YTZiNWM0ZDNlMmY",
		},
		{
			name:     "zero value cursor",
			cursor:   Cursor{Ms: 0, UID: uuid.Nil},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeCursor(tt.cursor)
			if got != tt.expected {
				t.Errorf("EncodeCursor() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name      string
		encoded   string
		wantMs    int64
		wantUID   uuid.UUID
		wantValid bool
	}{
		{
			name:      "valid cursor",
			encoded:   "MTczMDYzNTIwMDAwMHxjMWQ5YjdkYy1hMWIyLTRjM2QtOWU4Zi03YTZiNWM0ZDNlMmY",
			wantMs:    1730635200000,
			wantUID:   uuid.MustParse("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f"),
			wantValid: true,
		},
		{
			name:      "empty string",
			encoded:   "",
			wantMs:    0,
			wantUID:   uuid.Nil,
			wantValid: false,
		},
		{
			name:      "invalid base64",
			encoded:   "not-base64!!!",
			wantMs:    0,
			wantUID:   uuid.Nil,
			wantValid: false,
		},
		{
			name:      "invalid format (no pipe)",
			encoded:   "MTIzNDU2Nzg5MA", // "1234567890" base64
			wantMs:    0,
			wantUID:   uuid.Nil,
			wantValid: false,
		},
		{
			name:      "invalid timestamp",
			encoded:   "YWJjfGMxZDliN2RjLWExYjItNGMzZC05ZThmLTdhNmI1YzRkM2UyZg", // "abc|c1d9b7dc-..."
			wantMs:    0,
			wantUID:   uuid.Nil,
			wantValid: false,
		},
		{
			name:      "invalid uuid",
			encoded:   "MTIzNDU2fG5vdC1hLXV1aWQ", // "123456|not-a-uuid"
			wantMs:    0,
			wantUID:   uuid.Nil,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := DecodeCursor(tt.encoded)
			if valid != tt.wantValid {
				t.Errorf("DecodeCursor() valid = %v, want %v", valid, tt.wantValid)
			}
			if valid {
				if got.Ms != tt.wantMs {
					t.Errorf("DecodeCursor() Ms = %v, want %v", got.Ms, tt.wantMs)
				}
				if got.UID != tt.wantUID {
					t.Errorf("DecodeCursor() UID = %v, want %v", got.UID, tt.wantUID)
				}
			}
		})
	}
}

func TestCursorRoundTrip(t *testing.T) {
	original := Cursor{
		Ms:  1730635200000,
		UID: uuid.MustParse("c1d9b7dc-a1b2-4c3d-9e8f-7a6b5c4d3e2f"),
	}

	encoded := EncodeCursor(original)
	decoded, valid := DecodeCursor(encoded)

	if !valid {
		t.Fatal("DecodeCursor() failed for valid cursor")
	}
	if decoded.Ms != original.Ms {
		t.Errorf("Round trip Ms = %v, want %v", decoded.Ms, original.Ms)
	}
	if decoded.UID != original.UID {
		t.Errorf("Round trip UID = %v, want %v", decoded.UID, original.UID)
	}
}

func TestRFC3339(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{
			name: "normal timestamp",
			ms:   1730635200000,
			want: "2024-11-03T12:00:00Z",
		},
		{
			name: "epoch",
			ms:   0,
			want: "1970-01-01T00:00:00Z",
		},
		{
			name: "with milliseconds",
			ms:   1730635200123,
			want: "2024-11-03T12:00:00.123Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RFC3339(tt.ms)
			if got != tt.want {
				t.Errorf("RFC3339() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNowMs(t *testing.T) {
	before := NowMs()
	after := NowMs()

	if after < before {
		t.Error("NowMs() went backwards in time")
	}
	if after-before > 1000 {
		t.Errorf("NowMs() took more than 1 second between calls: %d ms", after-before)
	}
}
