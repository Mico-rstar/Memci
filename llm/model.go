package llm

import (
	"context"
	"fmt"
	"memci/config"
	"memci/logger"
	"memci/message"
	"memci/tools"

	"github.com/sashabaranov/go-openai"
)

const (
	ModelQwenMax   ModelName = "qwen-max"
	ModelQwenPlus  ModelName = "qwen-plus"
	ModelQwenFlash ModelName = "qwen-flash"
)

type ModelName string


type Model struct {
	client *openai.Client
	lg     logger.Logger
	name   ModelName
	tools  tools.ToolList
}

func NewModel(cfg config.Config, logger logger.Logger, name ModelName, tools tools.ToolList) *Model {
	oaiCfg := openai.DefaultConfig(cfg.LLM.ApiKey)
	oaiCfg.BaseURL = cfg.LLM.BaseUrl
	return &Model{
		client: openai.NewClientWithConfig(
			oaiCfg,
		),
		lg:    logger,
		name:  name,
		tools: tools,
	}
}

func (m *Model) Process(msgs message.MessageList) message.Message {
	rsp, err := m.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    "qwen-flash",
			Messages: msgs.Msgs,
			Tools:    m.tools.ConvertToOaiFormat(),
		},
	)
	if err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
		return message.Message{}
	}

	if len(rsp.Choices) == 0 {
		m.lg.Error("no response", logger.F("position", "llm.Model.Process"))
		return message.Message{}
	}

	msg := message.Message{
		Role:      message.Assistant,
		Content:   rsp.Choices[0].Message.Content,
		ToolCalls: rsp.Choices[0].Message.ToolCalls,
	}

	fmt.Printf("%#v\n", rsp)

	return msg
}
