package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Response represents a user's input from the terminal or an error.
type Response struct {
	Value string
	Err   error
}

// TerminalReader reads input from the terminal in a non-blocking way.
type TerminalReader struct {
	inputCh chan Response
}

// NewTerminalReader creates a new TerminalReader and starts the reading loop.
func NewTerminalReader(ctx context.Context, source io.Reader) *TerminalReader {
	tr := &TerminalReader{
		inputCh: make(chan Response),
	}
	go tr.readLoop(ctx, source)
	return tr
}

// readLoop continuously reads from the source and sends responses to the input channel.
func (tr *TerminalReader) readLoop(ctx context.Context, source io.Reader) {
	reader := bufio.NewReader(source) // os.Stdin
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		stdInput, err := reader.ReadString('\n')
		if err != nil {
			select {
			case tr.inputCh <- Response{Err: err}:
			case <-ctx.Done():
			}
			return
		}

		sentence := strings.TrimSpace(stdInput)
		if sentence == "" {
			fmt.Println("Please provide non empty answer")
			continue
		}

		select {
		case tr.inputCh <- Response{Value: sentence}:
		case <-ctx.Done():
			return
		}
	}
}

// Interactor displays a question to the user in the terminal and returns their input.
func (tr *TerminalReader) Interactor(ctx context.Context, input QuestionInput) (string, error) {
	fmt.Println(input.Question)
	if len(input.Choices) > 0 {
		for i, choice := range input.Choices {
			var suffix string
			if i == len(input.Choices)-1 {
				suffix = ""
			} else {
				suffix = ", "
			}
			fmt.Printf("%s%s\n", choice, suffix)
		}
		fmt.Println("")
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-ticker.C:
		return "", errors.New("Response was not provided in time")
	case res := <-tr.inputCh:
		return res.Value, res.Err
	}
}
