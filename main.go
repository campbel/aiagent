package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/campbel/yoshi"
	"github.com/charmbracelet/log"
	"github.com/sashabaranov/go-openai"
)

type Options struct {
	Config string `yoshi:"--config,-c;Config file to use"`
}

func main() {
	client := openai.NewClient(os.Getenv("OPENAI_KEY"))
	yoshi.New("aiagent").Run(func(options Options) {
		data, err := os.ReadFile(options.Config)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		var config Config
		if err := json.Unmarshal(data, &config); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}

		messages := []openai.ChatCompletionMessage{{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are an agent that is trying to get information from a customer. Ask only one question at a time and wait for the response.
			Here is the information we need you to collect:
			` + config.getPromptString() + `
			Once you have all of the information respond with "DONE" and then write a summary of the information you collected. 
			The summary should be in the format of "<key>: <value>" where the key is from the input we gave you and the value is what you collected. 
			Each item should be on its own line.`,
		}}

		reader := bufio.NewReader(os.Stdin)
		for i := 0; ; i++ {
			log.Info("starting iteration", "iteration", i)
			response, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model:     "gpt-3.5-turbo",
				MaxTokens: 100,
				Messages:  messages,
			})
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}

			agentMessage := response.Choices[0].Message.Content
			parts := strings.Split(agentMessage, "DONE")
			if len(parts) > 1 {
				fmt.Println(parts[0])
				fmt.Println(parseResults(strings.Split(parts[1], "\n")))
				return
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: agentMessage,
			})

			fmt.Println(agentMessage)

			userInput, err := reader.ReadString('\n')
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}

			fmt.Println("You endtered " + userInput)

			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: userInput,
			})
		}
	})
}

type Config struct {
	Prompt string
	Data   []struct {
		Key         string
		Description string
	}
}

func (c Config) getPromptString() string {
	var prompt string
	for _, d := range c.Data {
		prompt += d.Key + ": " + d.Description + "\n"
	}
	return prompt
}

type Results struct {
	Key   string
	Value string
}

func parseResults(vals []string) []Results {
	var results []Results
	for _, v := range vals {
		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			continue
		}
		results = append(results, Results{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}
	return results
}
