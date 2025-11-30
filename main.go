package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// main is the entry point of the application.
// It initializes the Genkit client, defines tools, and runs the agent loop.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	apIKey := os.Getenv("API_KEY")
	if apIKey == "" {
		log.Fatal("API key is required")
	}

	g := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{
		APIKey: apIKey,
	}),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)

	if g == nil {
		log.Fatal("can't init genkit")
	}

	DefineAskQuestionTool(g)

	// var systemPrompt SystemPrompt = "Ask clarifying questions until you have a complete solution. Provide a question with options."
	var systemPrompt SystemPrompt = `You are a helpful assistant that asks clarifying questions to gather information.

	CRITICAL INSTRUCTIONS:
	1. You MUST use the askQuestion tool for EVERY question - never ask questions directly in your response
	2. Continue asking questions until you have ALL necessary information
	3. If the user's answer is unclear or not in the provided options, ask a follow-up question using the askQuestion tool
	4. If the user provides an unexpected answer, acknowledge it and ask another clarifying question
	5. Only provide your final response when you have complete information about:
	   - The recipients (age, gender, interests)
	   - Budget constraints
	   - Any special preferences or restrictions

	Remember: ALWAYS use the askQuestion tool to interact with the user. Never stop until you have gathered all necessary details.`

	var userPrompt UserPrompt = "Please help with Christmas presents for children 8 and 11 years old children"
	toolNames := []string{"askQuestion"}
	generator := GenkitGenerator{AIClient: g}
	terminalReader := NewTerminalReader(ctx, os.Stdin)

	interruptionHandler := InterruptionHandler{
		generator:       &generator,
		UserInteraction: terminalReader.Interactor,
	}

	finalResponse, err := RunAgent(ctx, &Options{
		generator:       &generator,
		systemPrompt:    systemPrompt,
		userPrompt:      userPrompt,
		toolNames:       toolNames,
		responseHandler: &interruptionHandler,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Println(finalResponse)
}
