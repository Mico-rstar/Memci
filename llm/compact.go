package llm

import (
	"memci/config"
	"memci/logger"
	"memci/message"
	"memci/prompts"
	"memci/tools"
)

type CompactModel struct {
	Model
	sysPrompt	string
}

func NewCompactModel(cfg *config.Config, logger logger.Logger) *CompactModel {
	return &CompactModel{
		Model: *NewModel(cfg, logger, ModelName(cfg.LLM.CompressModel), *tools.NewToolList()),
		sysPrompt: prompts.SYS_PROMPT_COMPACT,
	}
}

func (c *CompactModel) Process(msgs message.MessageList) (message.Message, error) {
	compMsgs := message.NewMessageList()
	compMsgs.AddCachedMessage(message.System, c.sysPrompt)
	compMsgs.AddMessageList(&msgs)
	compMsgs.AddMessage(message.User, prompts.USR_PROMPT_COMPACT)
	return c.Model.Process(*compMsgs)
}
