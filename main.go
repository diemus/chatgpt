package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/sashabaranov/go-openai"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var conversation []openai.ChatCompletionMessage
var debug bool

func completer(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return nil
	}
	s := []prompt.Suggest{
		{Text: "exit", Description: "exit chatGPT"},
		{Text: "reset", Description: "reset conversation context"},
		{Text: "context", Description: "show conversation context"},
		{Text: "debug", Description: "enable or disable debug mode"},
	}
	return prompt.FilterHasPrefix(s, d.TextBeforeCursor(), true)
}

func executor(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	switch input {
	case "exit", "e":
		os.Exit(0)
	case "context":
		for i, data := range conversation {
			fmt.Printf("#%d %s: %s\n", i, data.Role, data.Content)
		}
		return
	case "reset":
		conversation = []openai.ChatCompletionMessage{}
		return
	case "debug":
		debug = !debug
		fmt.Printf("debug mode %v\n", debug)
		return
	}

	err := processQuestion(input)
	if err != nil {
		fmt.Printf("[ERROR]%s\n", err)
		os.Exit(1)
	}
}

func processQuestion(question string) error {
	token := os.Getenv("CHATGPT_API_KEY")
	proxy := os.Getenv("CHATGPT_API_PROXY")
	config := openai.DefaultConfig(token)

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			fmt.Printf("[ERROR]%s\n", err)
			os.Exit(1)
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
		config.HTTPClient = &http.Client{
			Transport: transport,
		}
	}

	conversation = append(conversation, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})

	client := openai.NewClientWithConfig(config)
	stream, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: conversation,
			Stream:   true,
		},
	)
	defer stream.Close()

	if err != nil {
		return err
	}

	answer := strings.Builder{}
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// 结束时换行
			fmt.Printf("\n")
			break
		}

		if err != nil {
			fmt.Printf("[ERROR]%s\n", err)
			return err
		}

		for _, choice := range response.Choices {
			answer.WriteString(choice.Delta.Content)
			fmt.Printf("%v", choice.Delta.Content)
		}
	}

	//将回复整体添加到下次请求的上下文中
	conversation = append(conversation, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: answer.String(),
	})

	return nil
}

func main() {
	fmt.Println("Ask a question(Press Ctrl+D to exit)")
	prompt.NewStdoutWriter()
	server := prompt.New(executor, completer,
		prompt.OptionPrefix(">>> "),
		prompt.OptionPrefixTextColor(prompt.Cyan),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionSuggestionTextColor(prompt.DarkGray),
		prompt.OptionDescriptionTextColor(prompt.DarkGray),
		prompt.OptionDescriptionBGColor(prompt.Cyan),
		prompt.OptionSelectedDescriptionTextColor(prompt.Black),
		prompt.OptionSelectedDescriptionBGColor(prompt.Turquoise),
	)
	server.Run()
	return
}
