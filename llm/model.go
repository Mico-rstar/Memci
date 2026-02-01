package llm

import (
	"context"
	"memci/config"
	"memci/logger"
	"memci/message"

	"github.com/sashabaranov/go-openai"
)

var (
	ModelQwenMax   ModelName = "qwen-max"
	ModelQwenPlus  ModelName = "qwen-plus"
	ModelQwenFlash ModelName = "qwen-flash"
)

type ModelName string

type Model struct {
	client *openai.Client
	lg     logger.Logger
	name   ModelName
}

func NewModel(cfg config.Config, logger logger.Logger, name ModelName) *Model {
	oaiCfg := openai.DefaultConfig(cfg.LLM.ApiKey)
	oaiCfg.BaseURL = cfg.LLM.BaseUrl
	return &Model{
		client: openai.NewClientWithConfig(
			oaiCfg,
		),
		lg:   logger,
		name: name,
	}
}

func (m *Model) Process(msgs message.MessageList) message.Message {
	rsp, err := m.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    "qwen-flash",
			Messages: msgs.Msgs,
		},
	)
	if err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
	}

	msg := message.Message{
		Role:    message.Assistant,
		Content: rsp.Choices[0].Message.Content,
	}
	return msg
}
