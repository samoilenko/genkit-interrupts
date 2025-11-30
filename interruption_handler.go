package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// getQuestionInput converts an arbitrary input into a QuestionInput struct.
// It handles type conversion from map[string]any to the structured QuestionInput type.
func getQuestionInput(input any) (*QuestionInput, error) {
	var questionInput QuestionInput
	rawInput, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected input type: %T", input)
	}

	jsonBytes, err := json.Marshal(rawInput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, &questionInput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	return &questionInput, nil
}

// UserInteractionFunc sends questions to the user and returns their answer.
type UserInteractionFunc func(ctx context.Context, input QuestionInput) (string, error)

// InterruptionHandler handles interruptions during AI generation, specifically for asking clarifying questions.
type InterruptionHandler struct {
	generator       Generator
	UserInteraction UserInteractionFunc
}

// handleResponse processes the model response, handling any "askQuestion" tool calls (interrupts).
// It prompts the user for input and continues generation until a final response is reached.
func (ih *InterruptionHandler) handleResponse(ctx context.Context, response *ai.ModelResponse) (*ai.ModelResponse, error) {
	askQuestion := ih.generator.LookupTool("askQuestion")
	if askQuestion == nil {
		return nil, errors.New("askQuestion tool not found")
	}

	var err error
	for response.FinishReason == "interrupted" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var answers []*ai.Part
		// multiple interrupts can be called at once, so we handle them all
		for _, part := range response.Interrupts() {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// convert map[string]any to QuestionInput
			questionInput, err := getQuestionInput(part.ToolRequest.Input)
			if err != nil {
				return nil, err
			}
			answer, err := ih.UserInteraction(ctx, *questionInput)
			if err != nil {
				return nil, err
			}
			// use the `Respond` method on our tool to populate answers
			answers = append(answers, askQuestion.Respond(part, any(answer), nil))
		}

		response, err = ih.generator.Generate(ctx,
			ai.WithMessages(response.History()...),
			ai.WithTools(askQuestion),
			ai.WithToolResponses(answers...),
		)

		if err != nil {
			return nil, err
		}
	}

	return response, nil
}
