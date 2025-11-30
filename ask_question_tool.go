package main

import (
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// QuestionInput contains a question to ask the user and optional multiple choice answers.
type QuestionInput struct {
	Question string   `json:"question" jsonschema:"description=A clarifying question"`
	Choices  []string `json:"choices" jsonschema:"description=the choices to display to the user"`
}

// DefineAskQuestionTool defines the "askQuestion" tool in the Genkit instance.
// This tool allows the AI to ask clarifying questions to the user.
func DefineAskQuestionTool(g *genkit.Genkit) {
	genkit.DefineTool(
		g,
		"askQuestion",
		"use this to ask the user any clarifying question",
		func(ctx *ai.ToolContext, input QuestionInput) (string, error) {
			return "", ctx.Interrupt(&ai.InterruptOptions{
				Metadata: map[string]any{
					"question": input,
				},
			})
		},
	)

}
