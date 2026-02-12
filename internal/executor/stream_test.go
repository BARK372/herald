package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamLine_WhenSystemInit_ExtractsSessionID(t *testing.T) {
	t.Parallel()

	line := `{"type":"system","subtype":"init","session_id":"ses_abc123","tools":["Read","Write"]}`

	event, err := ParseStreamLine([]byte(line))
	require.NoError(t, err)
	assert.Equal(t, "system", event.Type)
	assert.Equal(t, "init", event.Subtype)
	assert.Equal(t, "ses_abc123", event.SessionID)
}

func TestParseStreamLine_WhenAssistantText_ExtractsContent(t *testing.T) {
	t.Parallel()

	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll fix the auth bug."}]}}`

	event, err := ParseStreamLine([]byte(line))
	require.NoError(t, err)
	assert.Equal(t, "assistant", event.Type)
	require.NotNil(t, event.Message)
	require.Len(t, event.Message.Content, 1)
	assert.Equal(t, "text", event.Message.Content[0].Type)
	assert.Equal(t, "I'll fix the auth bug.", event.Message.Content[0].Text)
}

func TestParseStreamLine_WhenToolUse_ExtractsToolName(t *testing.T) {
	t.Parallel()

	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Write","input":{"file_path":"auth.go"}}]}}`

	event, err := ParseStreamLine([]byte(line))
	require.NoError(t, err)
	require.NotNil(t, event.Message)
	require.Len(t, event.Message.Content, 1)
	assert.Equal(t, "tool_use", event.Message.Content[0].Type)
	assert.Equal(t, "Write", event.Message.Content[0].Name)
}

func TestParseStreamLine_WhenResultSuccess_ExtractsCostAndTurns(t *testing.T) {
	t.Parallel()

	line := `{"type":"result","subtype":"success","session_id":"ses_abc","cost_usd":0.34,"duration_ms":45000,"num_turns":5}`

	event, err := ParseStreamLine([]byte(line))
	require.NoError(t, err)
	assert.Equal(t, "result", event.Type)
	assert.Equal(t, "success", event.Subtype)
	assert.InDelta(t, 0.34, event.CostUSD, 0.001)
	assert.Equal(t, int64(45000), event.Duration)
	assert.Equal(t, 5, event.NumTurns)
	assert.Equal(t, "ses_abc", event.SessionID)
}

func TestParseStreamLine_WhenResultFailure_HasSubtype(t *testing.T) {
	t.Parallel()

	line := `{"type":"result","subtype":"error","cost_usd":0.12,"duration_ms":5000,"num_turns":2}`

	event, err := ParseStreamLine([]byte(line))
	require.NoError(t, err)
	assert.Equal(t, "result", event.Type)
	assert.Equal(t, "error", event.Subtype)
}

func TestParseStreamLine_WhenMalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	line := `{this is not json}`

	event, err := ParseStreamLine([]byte(line))
	assert.Nil(t, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing stream event")
}

func TestParseStreamLine_WhenEmptyLine_ReturnsNil(t *testing.T) {
	t.Parallel()

	event, err := ParseStreamLine([]byte(""))
	assert.Nil(t, event)
	assert.NoError(t, err)
}

func TestExtractProgress_WhenTextContent_ReturnsTruncated(t *testing.T) {
	t.Parallel()

	event := &StreamEvent{
		Message: &StreamMessage{
			Content: []ContentBlock{
				{Type: "text", Text: "I'll fix this bug now."},
			},
		},
	}

	progress := ExtractProgress(event)
	assert.Equal(t, "I'll fix this bug now.", progress)
}

func TestExtractProgress_WhenToolUse_ReturnsToolName(t *testing.T) {
	t.Parallel()

	event := &StreamEvent{
		Message: &StreamMessage{
			Content: []ContentBlock{
				{Type: "tool_use", Name: "Edit"},
			},
		},
	}

	progress := ExtractProgress(event)
	assert.Equal(t, "Using tool: Edit", progress)
}

func TestExtractProgress_WhenNoMessage_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	event := &StreamEvent{Type: "system"}
	assert.Empty(t, ExtractProgress(event))
}

func TestExtractOutput_CollectsAllTextBlocks(t *testing.T) {
	t.Parallel()

	event := &StreamEvent{
		Message: &StreamMessage{
			Content: []ContentBlock{
				{Type: "text", Text: "Part 1. "},
				{Type: "tool_use", Name: "Read"},
				{Type: "text", Text: "Part 2."},
			},
		},
	}

	output := ExtractOutput(event)
	assert.Equal(t, "Part 1. Part 2.", output)
}
