package main

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"os"
	"strings"
)

var msg []map[string]string
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
		for i, data := range msg {
			fmt.Printf("#%d %s: %s\n", i, data["role"], data["content"])
		}
		return
	case "reset":
		msg = []map[string]string{}
		return
	case "debug":
		debug = !debug
		fmt.Printf("debug mode %v\n", debug)
		return
	}

	answer, err := processQuestion(input)
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(strings.TrimSpace(answer["content"]))
}

func processQuestion(question string) (map[string]string, error) {
	restyClient := resty.New()

	key := os.Getenv("CHATGPT_API_KEY")
	proxy := os.Getenv("CHATGPT_API_PROXY")

	if proxy != "" {
		restyClient.SetProxy(proxy)
	}

	msg = append(msg, map[string]string{"role": "user", "content": question})
	resp, err := restyClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key)).
		SetBody(map[string]interface{}{
			"model":    "gpt-3.5-turbo",
			"messages": msg,
		}).
		Post("https://api.openai.com/v1/chat/completions")
	if err != nil {
		return nil, err
	}

	if debug {
		fmt.Println(resp.String())
		fmt.Println("-------------------------")
	}

	result := gjson.Get(resp.String(), "choices.0.message.content")

	reply := map[string]string{"role": "assistant", "content": result.String()}

	msg = append(msg, reply)

	return reply, nil
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
