package main

import (
	"fmt"
	"memci/config"
	"memci/llm"
	"memci/logger"
	"memci/message"
	"memci/tools"
)

func main() {
	cfg := config.LoadConfig(".env")
	lg := logger.NewNoOpLogger()

	model := llm.NewModel(cfg, lg, llm.ModelQwenFlash, *tools.NewToolList())

	msgs := message.NewMessageList().
		AddMessage(message.User, "你好")

	resp, err := model.Process(*msgs)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(resp.Content)
}
