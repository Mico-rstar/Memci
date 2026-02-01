package main

import (
	"context"
	"fmt"
	"memci/config"

	openai "github.com/sashabaranov/go-openai"
)



func main() {
	cfg := config.LoadConfig(".env")
	oaiCfg := openai.DefaultConfig(cfg.LLM.ApiKey)
	oaiCfg.BaseURL = cfg.LLM.BaseUrl
	client := openai.NewClientWithConfig(oaiCfg)

	rsp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "qwen-flash",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: "你好",
				},
			},
		},
	)

	


	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(rsp.Choices[0].Message.Content)

	// msg := message.Message{}
	
}
