package main

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversationLoopHandler_HandleResponse(t *testing.T) {
	// Helper to create a simple text response
	simpleResponse := createTextResponse("Hello", "stop")

	t.Run("Conversation finishes immediately", func(t *testing.T) {
		mockGen := NewMockGenerator(
			[]*ai.ModelResponse{}, // No extra generations needed
			map[string]ai.Tool{"askQuestion": createMockTool("askQuestion")},
		)
		// GenerateBool returns true (finished) immediately
		mockGen.boolResponses = []bool{true}

		handler := &ConversationLoopHandler{
			generator:           mockGen,
			validationPrompt:    "Is finished?",
			interruptionHandler: InterruptionHandler{generator: mockGen},
		}

		// Mock InterruptionHandler to just return the response
		// In a real scenario, InterruptionHandler might do more, but here we assume it passes through if no interrupts
		// We need to make sure InterruptionHandler.handleResponse is called.
		// Since InterruptionHandler struct is used directly, we can't easily mock it unless we change the struct to use an interface.
		// However, InterruptionHandler logic is: if interrupted, handle it. If not, return response.
		// So passing a non-interrupted response should be fine.

		ctx := context.Background()
		resp, err := handler.handleResponse(ctx, simpleResponse)

		require.NoError(t, err)
		assert.Equal(t, simpleResponse, resp)
		assert.Equal(t, 1, mockGen.boolCallIndex)
	})

	t.Run("Conversation loops once", func(t *testing.T) {
		// Initial response -> Loop check (false) -> Generate new response (with prompt from user) -> Loop check (true)

		// We need to mock UserInteraction for the InterruptionHandler used inside ConversationLoopHandler
		// But wait, ConversationLoopHandler uses `cv.interruptionHandler.UserInteraction`
		// `InterruptionHandler` struct has `UserInteraction` field.

		mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
			return "User Answer", nil
		}

		mockGen := NewMockGenerator(
			[]*ai.ModelResponse{
				// Response generated inside the loop
				createTextResponse("Final Answer", "stop"),
			},
			map[string]ai.Tool{"askQuestion": createMockTool("askQuestion")},
		)
		// 1. First check: false (not finished)
		// 2. Second check: true (finished)
		mockGen.boolResponses = []bool{false, true}

		handler := &ConversationLoopHandler{
			generator:        mockGen,
			validationPrompt: "Is finished?",
			interruptionHandler: InterruptionHandler{
				generator:       mockGen,
				UserInteraction: mockUserInteraction,
			},
		}

		ctx := context.Background()
		resp, err := handler.handleResponse(ctx, simpleResponse)

		require.NoError(t, err)
		assert.Equal(t, "Final Answer", resp.Text())
		assert.Equal(t, 2, mockGen.boolCallIndex)
		assert.Equal(t, 1, mockGen.callIndex) // One generation call inside the loop
	})

	t.Run("Tool not found error", func(t *testing.T) {
		mockGen := NewMockGenerator(
			[]*ai.ModelResponse{},
			map[string]ai.Tool{}, // No tools
		)

		handler := &ConversationLoopHandler{
			generator:           mockGen,
			validationPrompt:    "Is finished?",
			interruptionHandler: InterruptionHandler{generator: mockGen},
		}

		ctx := context.Background()
		_, err := handler.handleResponse(ctx, simpleResponse)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "askQuestion tool not found")
	})
}
