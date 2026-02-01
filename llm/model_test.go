package llm

import (
	"fmt"
	"memci/config"
	"memci/logger"
	"memci/message"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()

	msgs := message.NewMessageList()
	msgs.AddMessage(message.User, "你好")

	model := NewModel(*cfg, lg, ModelQwenFlash)
	rsp := model.Process(*msgs)

	require.NotNil(t, rsp.Content)
	require.Equal(t, message.Assistant, rsp.Role)

	fmt.Println(rsp)
}

