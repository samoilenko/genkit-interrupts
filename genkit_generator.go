package main

import (
	"context"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// GenkitGenerator is a wrapper around genkit.Genkit that implements the Generator interface.
type GenkitGenerator struct {
	AIClient *genkit.Genkit
}

// Generate generates a response from the AI model using the provided options.
func (g *GenkitGenerator) Generate(ctx context.Context, opts ...ai.GenerateOption) (*ai.ModelResponse, error) {
	return genkit.Generate(ctx, g.AIClient, opts...)
}

// LookupTool looks up a tool by name in the Genkit instance.
func (g *GenkitGenerator) LookupTool(name string) ai.Tool {
	return genkit.LookupTool(g.AIClient, name)
}

// GenerateBool generates a boolean response from the AI model based on the prompt and history.
func (g *GenkitGenerator) GenerateBool(ctx context.Context, prompt string, history []*ai.Message) (bool, error) {
	result, _, err := genkit.GenerateData[bool](ctx, g.AIClient,
		ai.WithMessages(history...),
		ai.WithSystem(prompt),
	)

	if err != nil {
		return false, err
	}

	return *result, nil
}
