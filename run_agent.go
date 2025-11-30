package main

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// ResponseHandler defines the interface for handling model responses, potentially involving interruptions.
type ResponseHandler interface {
	handleResponse(ctx context.Context, response *ai.ModelResponse) (*ai.ModelResponse, error)
}

// SystemPrompt represents the system-level instructions for the AI model.
type SystemPrompt string

// UserPrompt represents the user's input prompt for the AI model.
type UserPrompt string

// Generator represents an AI model that can generate responses and access tools.
type Generator interface {
	Generate(ctx context.Context, opts ...ai.GenerateOption) (*ai.ModelResponse, error)
	LookupTool(name string) ai.Tool
	GenerateBool(ctx context.Context, prompt string, history []*ai.Message) (bool, error)
}

// Options contains the configuration for running the agent.
type Options struct {
	generator       Generator
	systemPrompt    SystemPrompt
	userPrompt      UserPrompt
	toolNames       []string
	responseHandler ResponseHandler
}

// RunAgent communicates with a user to ask clarifying questions during AI generation.
// It uses the askQuestion tool to interrupt the generation process, collect user input,
// and continue generation with the provided answers until a final response is produced.
func RunAgent(
	ctx context.Context,
	options *Options,
) (string, error) {
	tools := make([]ai.ToolRef, 0, len(options.toolNames))
	for _, toolName := range options.toolNames {
		tool := options.generator.LookupTool(toolName)
		if tool == nil {
			return "", fmt.Errorf("%s tool not found", toolName)
		}
		tools = append(tools, tool)
	}

	response, err := options.generator.Generate(ctx,
		ai.WithPrompt(string(options.userPrompt)),
		ai.WithSystem(string(options.systemPrompt)),
		ai.WithTools(tools...),
	)
	if err != nil {
		return "", err
	}

	if options.responseHandler == nil {
		return response.Text(), nil
	}

	response, err = options.responseHandler.handleResponse(ctx, response)
	if err != nil {
		return "", err
	}

	return response.Text(), nil
}
