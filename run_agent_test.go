package main

import (
	"context"
	"errors"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockGenerateCall struct {
	Options       []ai.GenerateOption
	HasHistory    bool
	HasTools      bool
	ToolResponses int
}

// MockGenerator simulates the genkit.Generate function with predefined responses
type MockGenerator struct {
	responses      []*ai.ModelResponse
	callIndex      int
	capturedCalls  []MockGenerateCall
	tools          map[string]ai.Tool
	messageHistory []*ai.Message
	boolResponses  []bool
	boolCallIndex  int
}

func (m *MockGenerator) Generate(ctx context.Context, opts ...ai.GenerateOption) (*ai.ModelResponse, error) {
	if m.callIndex >= len(m.responses) {
		return nil, errors.New("no more mock responses available")
	}

	// Capture call details for assertions
	call := MockGenerateCall{
		Options: opts,
	}
	m.capturedCalls = append(m.capturedCalls, call)

	response := m.responses[m.callIndex]
	m.callIndex++

	// Update the response's Request.Messages to include the message history
	// This simulates how the real generator maintains conversation context
	if response.Request == nil {
		response.Request = &ai.ModelRequest{
			Messages: m.messageHistory,
		}
	} else {
		response.Request.Messages = append([]*ai.Message{}, m.messageHistory...)
	}

	// Add the current response message to history for the next call
	if response.Message != nil {
		m.messageHistory = append(m.messageHistory, response.Message)
	}

	return response, nil
}

func (m *MockGenerator) GenerateBool(ctx context.Context, prompt string, history []*ai.Message) (bool, error) {
	if m.boolCallIndex >= len(m.boolResponses) {
		// Default to true if no more responses are defined, to avoid infinite loops in tests
		return true, nil
	}
	response := m.boolResponses[m.boolCallIndex]
	m.boolCallIndex++
	return response, nil
}

func (m *MockGenerator) LookupTool(name string) ai.Tool {
	return m.tools[name]
}

func NewMockGenerator(responses []*ai.ModelResponse, tools map[string]ai.Tool) *MockGenerator {
	return &MockGenerator{
		responses:      responses,
		callIndex:      0,
		tools:          tools,
		capturedCalls:  make([]MockGenerateCall, 0),
		messageHistory: make([]*ai.Message, 0),
		boolResponses:  []bool{},
		boolCallIndex:  0,
	}
}

// MockTool implements ai.Tool interface for testing
type MockTool struct {
	name        string
	description string
}

func (mt *MockTool) Name() string {
	return mt.name
}

func (mt *MockTool) Definition() *ai.ToolDefinition {
	return &ai.ToolDefinition{
		Name:        mt.name,
		Description: mt.description,
	}
}

func (mt *MockTool) RunRaw(ctx context.Context, input any) (any, error) {
	// For testing, we don't actually run the tool
	// The userInteraction function handles the actual interaction
	return nil, nil
}

func (mt *MockTool) Respond(toolReq *ai.Part, outputData any, opts *ai.RespondOptions) *ai.Part {
	return &ai.Part{
		ToolResponse: &ai.ToolResponse{
			Name:   mt.name,
			Output: outputData,
		},
	}
}

func (mt *MockTool) Restart(toolReq *ai.Part, opts *ai.RestartOptions) *ai.Part {
	// For testing, we can return a simple restart part
	return &ai.Part{
		ToolRequest: &ai.ToolRequest{
			Name:  mt.name,
			Input: toolReq.ToolRequest.Input,
		},
	}
}

func (mt *MockTool) Register(r api.Registry) {
	// For testing, we don't need to register with a real registry
}

// Helper function to create a mock tool
func createMockTool(name string) ai.Tool {
	return &MockTool{
		name:        name,
		description: "Mock tool for testing",
	}
}

// Helper function to create a tool request part
func createToolRequestPart(name string, question string, choices []string) *ai.Part {
	return &ai.Part{
		ToolRequest: &ai.ToolRequest{
			Name: name,
			Input: map[string]any{
				"question": question,
				"choices":  choices,
			},
		},
		Kind:     ai.PartToolRequest,
		Metadata: map[string]any{"interrupt": "interruptTest"},
	}
}

// Helper function to create a text response
func createTextResponse(text string, finishReason string) *ai.ModelResponse {
	return &ai.ModelResponse{
		Message: &ai.Message{
			Content: []*ai.Part{
				{Text: text},
			},
			Role: ai.RoleModel,
		},
		Request: &ai.ModelRequest{
			Messages: []*ai.Message{},
		},
		FinishReason: ai.FinishReason(finishReason),
	}
}

// Helper function to create an interrupted response with tool calls
func createInterruptedResponse(toolRequests ...*ai.Part) *ai.ModelResponse {
	return &ai.ModelResponse{
		Message: &ai.Message{
			Content: toolRequests,
			Role:    ai.RoleModel,
		},
		Request: &ai.ModelRequest{
			Messages: []*ai.Message{},
		},
		FinishReason: "interrupted",
	}
}

// TestInterruption_SimpleFlow tests a basic conversation with one interruption
func TestInterruption_SimpleFlow(t *testing.T) {
	mockResponses := []string{"Boy"}
	responseIndex := 0

	mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
		assert.Less(t, responseIndex, len(mockResponses), "unexpected question asked")
		response := mockResponses[responseIndex]
		responseIndex++

		// Verify the question structure
		assert.NotEmpty(t, input.Question)
		return response, nil
	}

	mockTool := createMockTool("askQuestion")
	tools := map[string]ai.Tool{
		"askQuestion": mockTool,
	}

	// Setup mock Genkit with predefined responses
	mockGen := NewMockGenerator(
		[]*ai.ModelResponse{
			// First call: AI asks a question
			createInterruptedResponse(
				createToolRequestPart(
					"askQuestion",
					"What gender are the children?",
					[]string{"Boy", "Girl", "Both"},
				),
			),
			// Second call: AI provides final answer
			createTextResponse(
				"Based on your answer, I recommend LEGO sets and science kits.",
				"stop",
			),
		},
		tools,
	)

	ctx := context.Background()

	result, err := RunAgent(
		ctx,
		&Options{
			generator: mockGen,
			responseHandler: &InterruptionHandler{
				generator:       mockGen,
				UserInteraction: mockUserInteraction,
			},
		},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "recommend")
	assert.Equal(t, 1, responseIndex, "all mock responses should be used")
	assert.Equal(t, 2, mockGen.callIndex, "should make 2 AI calls")
}

// TestInterruption_MultipleSimultaneousInterrupts tests handling multiple tool calls at once
func TestInterruption_MultipleSimultaneousInterrupts(t *testing.T) {
	answers := map[string]string{
		"What gender are the children?": "Boy and Girl",
		"What are their ages?":          "8 and 11",
	}
	questionsAsked := make(map[string]bool)

	mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
		answer, ok := answers[input.Question]
		require.True(t, ok, "unexpected question: %s", input.Question)
		questionsAsked[input.Question] = true
		return answer, nil
	}

	mockTool := createMockTool("askQuestion")
	tools := map[string]ai.Tool{
		"askQuestion": mockTool,
	}

	mockGen := NewMockGenerator(
		[]*ai.ModelResponse{
			// Multiple interrupts in one response
			createInterruptedResponse(
				createToolRequestPart(
					"askQuestion",
					"What gender are the children?",
					[]string{"Boy", "Girl", "Both"},
				),
				createToolRequestPart(
					"askQuestion",
					"What are their ages?",
					[]string{"5-7", "8-10", "11-13", "14+"},
				),
			),
			// Final response
			createTextResponse(
				"Based on both genders and ages...",
				"stop",
			),
		},
		tools,
	)

	ctx := context.Background()
	result, err := RunAgent(
		ctx,
		&Options{
			generator: mockGen,
			responseHandler: &InterruptionHandler{
				generator:       mockGen,
				UserInteraction: mockUserInteraction,
			},
		},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "Based on")
	assert.Equal(t, 2, mockGen.callIndex)
	assert.Equal(t, 2, len(questionsAsked), "both questions should be asked")
}

// // TestInterruption_ContextCancellation tests handling of context cancellation
func TestInterruption_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
		t.Fatal("should not be called when context is cancelled")
		return "", nil
	}

	mockTool := createMockTool("askQuestion")
	tools := map[string]ai.Tool{
		"askQuestion": mockTool,
	}

	mockGen := NewMockGenerator(
		[]*ai.ModelResponse{
			createInterruptedResponse(
				createToolRequestPart(
					"askQuestion",
					"What gender?",
					[]string{"Boy", "Girl"},
				),
			),
		},
		tools,
	)

	_, err := RunAgent(
		ctx,
		&Options{
			generator: mockGen,
			responseHandler: &InterruptionHandler{
				generator:       mockGen,
				UserInteraction: mockUserInteraction,
			},
		},
	)

	assert.ErrorIs(t, err, context.Canceled)
}

// TestInterruption_ToolNotFound tests error when tool is not available
func TestInterruption_ToolNotFound(t *testing.T) {
	mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
		t.Fatal("should not be called when tool is not found")
		return "", nil
	}

	// Empty tools map - tool lookup will fail
	mockGen := NewMockGenerator(
		[]*ai.ModelResponse{},
		map[string]ai.Tool{},
	)

	ctx := context.Background()
	_, err := RunAgent(
		ctx,
		&Options{
			generator: mockGen,
			responseHandler: &InterruptionHandler{
				generator:       mockGen,
				UserInteraction: mockUserInteraction,
			},
			toolNames: []string{"askQuestion"},
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "askQuestion tool not found")
}

// TestInterruption_ErrorInGenerate tests error handling
func TestInterruption_ErrorInGenerate(t *testing.T) {
	mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
		return "Boy", nil
	}

	mockTool := createMockTool("askQuestion")
	tools := map[string]ai.Tool{
		"askQuestion": mockTool,
	}

	mockGen := NewMockGenerator(
		[]*ai.ModelResponse{
			createInterruptedResponse(
				createToolRequestPart(
					"askQuestion",
					"What gender?",
					[]string{"Boy", "Girl"},
				),
			),
			// Missing second response - will cause error
		},
		tools,
	)

	ctx := context.Background()
	_, err := RunAgent(
		ctx,
		&Options{
			generator: mockGen,
			responseHandler: &InterruptionHandler{
				generator:       mockGen,
				UserInteraction: mockUserInteraction,
			},
			toolNames: []string{"askQuestion"},
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no more mock responses")
}

// Table-driven tests for different conversation scenarios
func TestInterruption_Scenarios(t *testing.T) {
	scenarios := []struct {
		name             string
		systemPrompt     SystemPrompt
		userPrompt       UserPrompt
		mockResponses    []string
		aiResponses      []*ai.ModelResponse
		expectedCalls    int
		expectedInResult string
	}{
		{
			name:          "quick resolution",
			systemPrompt:  "Be helpful",
			userPrompt:    "Help with gifts",
			mockResponses: []string{"Boy"},
			aiResponses: []*ai.ModelResponse{
				createInterruptedResponse(
					createToolRequestPart("askQuestion", "Gender?", []string{"Boy", "Girl"}),
				),
				createTextResponse("I recommend LEGO", "stop"),
			},
			expectedCalls:    2,
			expectedInResult: "recommend",
		},
		{
			name:          "detailed inquiry",
			systemPrompt:  "Ask detailed questions",
			userPrompt:    "Help with gifts",
			mockResponses: []string{"Boy", "Sports", "$50"},
			aiResponses: []*ai.ModelResponse{
				createInterruptedResponse(
					createToolRequestPart("askQuestion", "Gender?", []string{"Boy", "Girl"}),
				),
				createInterruptedResponse(
					createToolRequestPart("askQuestion", "Interests?", []string{"Sports", "Games"}),
				),
				createInterruptedResponse(
					createToolRequestPart("askQuestion", "Budget?", []string{"$50", "$100"}),
				),
				createTextResponse("Personalized recommendations", "stop"),
			},
			expectedCalls:    4,
			expectedInResult: "Personalized",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			responseIndex := 0
			mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
				if responseIndex >= len(scenario.mockResponses) {
					t.Fatal("unexpected question")
				}
				response := scenario.mockResponses[responseIndex]
				responseIndex++
				return response, nil
			}

			mockTool := createMockTool("askQuestion")
			tools := map[string]ai.Tool{
				"askQuestion": mockTool,
			}

			mockGen := NewMockGenerator(scenario.aiResponses, tools)

			ctx := context.Background()
			result, err := RunAgent(
				ctx,
				&Options{
					generator: mockGen,
					responseHandler: &InterruptionHandler{
						generator:       mockGen,
						UserInteraction: mockUserInteraction,
					},
					toolNames: []string{"askQuestion"},
				},
			)

			require.NoError(t, err)
			assert.Contains(t, result, scenario.expectedInResult)
			assert.Equal(t, len(scenario.mockResponses), responseIndex)
			assert.Equal(t, scenario.expectedCalls, mockGen.callIndex)
		})
	}
}

func TestRunAgent_WithConversationLoop(t *testing.T) {
	mockUserInteraction := func(ctx context.Context, input QuestionInput) (string, error) {
		return "User Answer", nil
	}

	mockTool := createMockTool("askQuestion")
	tools := map[string]ai.Tool{
		"askQuestion": mockTool,
	}

	mockGen := NewMockGenerator(
		[]*ai.ModelResponse{
			// 1. Initial response from RunAgent
			createTextResponse("Initial Question", "stop"),
			// 2. Response from the loop
			createTextResponse("Final Answer", "stop"),
		},
		tools,
	)
	// 1. First check: false (not finished)
	// 2. Second check: true (finished)
	mockGen.boolResponses = []bool{false, true}

	ctx := context.Background()

	interruptionHandler := InterruptionHandler{
		generator:       mockGen,
		UserInteraction: mockUserInteraction,
	}

	conversationLoopHandler := &ConversationLoopHandler{
		generator:           mockGen,
		validationPrompt:    "Is finished?",
		interruptionHandler: interruptionHandler,
	}

	result, err := RunAgent(
		ctx,
		&Options{
			generator:       mockGen,
			responseHandler: conversationLoopHandler,
			toolNames:       []string{"askQuestion"},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Final Answer", result)
	assert.Equal(t, 2, mockGen.callIndex)
	assert.Equal(t, 2, mockGen.boolCallIndex)
}
