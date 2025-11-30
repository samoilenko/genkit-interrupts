package main

import (
	"context"
	"errors"

	"github.com/firebase/genkit/go/ai"
)

type ConversationLoopHandler struct {
	generator           Generator
	validationPrompt    string
	interruptionHandler InterruptionHandler
}

func (cv *ConversationLoopHandler) handleResponse(ctx context.Context, response *ai.ModelResponse) (*ai.ModelResponse, error) {
	askQuestion := cv.generator.LookupTool("askQuestion")
	if askQuestion == nil {
		return nil, errors.New("askQuestion tool not found")
	}

	var err error
	var hasMoreQuestions bool = true
	for hasMoreQuestions {
		response, err = cv.interruptionHandler.handleResponse(ctx, response)
		if err != nil {
			return nil, err
		}

		isConversationFinished, err := cv.generator.GenerateBool(ctx,
			cv.validationPrompt,
			response.History(),
		)

		if err != nil {
			return nil, err
		}

		hasMoreQuestions = !isConversationFinished
		if hasMoreQuestions {
			response, err = cv.generator.Generate(ctx,
				ai.WithMessages(response.History()...),
				ai.WithTools(askQuestion),
				ai.WithPrompt(cv.interruptionHandler.UserInteraction(ctx, QuestionInput{Question: response.Text()})),
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return response, nil
}
