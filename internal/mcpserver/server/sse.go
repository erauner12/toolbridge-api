package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEStream manages an SSE connection for a session
type SSEStream struct {
	mu        sync.Mutex
	w         http.ResponseWriter
	flusher   http.Flusher
	eventID   int
	sessionID string
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewSSEStream creates a new SSE stream
func NewSSEStream(ctx context.Context, w http.ResponseWriter, sessionID string) (*SSEStream, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	streamCtx, cancel := context.WithCancel(ctx)

	return &SSEStream{
		w:         w,
		flusher:   flusher,
		sessionID: sessionID,
		ctx:       streamCtx,
		cancel:    cancel,
	}, nil
}

// SendMessage sends a JSON-RPC message via SSE
func (s *SSEStream) SendMessage(msg interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventID++

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write SSE event
	fmt.Fprintf(s.w, "event: message\n")
	fmt.Fprintf(s.w, "id: %d\n", s.eventID)
	fmt.Fprintf(s.w, "data: %s\n\n", data)

	s.flusher.Flush()
	return nil
}

// Close closes the SSE stream
func (s *SSEStream) Close() {
	s.cancel()
}

// Done returns a channel that's closed when the stream is closed
func (s *SSEStream) Done() <-chan struct{} {
	return s.ctx.Done()
}
