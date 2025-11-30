package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetQuestionInput tests the input parsing function
func TestGetQuestionInput(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    QuestionInput
		expectError bool
	}{
		{
			name: "valid input with choices",
			input: map[string]any{
				"question": "What gender?",
				"choices":  []any{"Boy", "Girl", "Both"},
			},
			expected: QuestionInput{
				Question: "What gender?",
				Choices:  []string{"Boy", "Girl", "Both"},
			},
			expectError: false,
		},
		{
			name: "valid input minimal",
			input: map[string]any{
				"question": "Pick one",
				"choices":  []any{"A", "B"},
			},
			expected: QuestionInput{
				Question: "Pick one",
				Choices:  []string{"A", "B"},
			},
			expectError: false,
		},
		{
			name:        "invalid input type",
			input:       "not a map",
			expectError: true,
		},
		{
			name:        "nil input",
			input:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getQuestionInput(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Question, result.Question)
				assert.ElementsMatch(t, tt.expected.Choices, result.Choices)
			}
		})
	}
}
