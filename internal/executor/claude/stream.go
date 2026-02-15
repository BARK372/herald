package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

// StreamEvent represents a single line from Claude Code's stream-json output.
type StreamEvent struct {
	Type      string         `json:"type"`
	Subtype   string         `json:"subtype,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Message   *StreamMessage `json:"message,omitempty"`
	CostUSD   float64        `json:"cost_usd,omitempty"`
	Duration  int64          `json:"duration_ms,omitempty"`
	NumTurns  int            `json:"num_turns,omitempty"`
}

// StreamMessage wraps the assistant's message in a stream event.
type StreamMessage struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock is a piece of content in an assistant message.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ParseStreamLine parses a single line of stream-json output.
func ParseStreamLine(line []byte) (*StreamEvent, error) {
	if len(line) == 0 {
		return nil, nil
	}

	var event StreamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		if syntaxErr, ok := errors.AsType[*json.SyntaxError](err); ok {
			slog.Debug("malformed JSON in stream",
				"offset", syntaxErr.Offset,
				"line_preview", truncateBytes(line, 100))
		}
		return nil, fmt.Errorf("parsing stream event: %w", err)
	}

	return &event, nil
}

// ExtractProgress returns a human-readable progress hint from an event.
func ExtractProgress(event *StreamEvent) string {
	if event.Message == nil {
		return ""
	}

	for _, block := range event.Message.Content {
		switch block.Type {
		case "text":
			return truncateStr(block.Text, 200)
		case "tool_use":
			return fmt.Sprintf("Using tool: %s", block.Name)
		}
	}

	return ""
}

// ExtractOutput collects all text content from an event.
func ExtractOutput(event *StreamEvent) string {
	if event.Message == nil {
		return ""
	}

	var out string
	for _, block := range event.Message.Content {
		if block.Type == "text" {
			out += block.Text
		}
	}
	return out
}

func truncateBytes(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
