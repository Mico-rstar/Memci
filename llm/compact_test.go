package llm

import (
	"encoding/json"
	"fmt"
	"memci/config"
	"memci/logger"
	"memci/message"
	"os"
	"path/filepath"
	"testing"

	"memci/test"

	"github.com/stretchr/testify/require"
)

func TestCompactModel(t *testing.T) {

	cfg := config.LoadConfig("../config/config.toml")
	lg := logger.NewNoOpLogger()
	cpml := NewCompactModel(cfg, lg)

	// 读取数据集
	datasetPath := filepath.Join("..", "test", "dataset.json")
	datasetFile, err := os.ReadFile(datasetPath)
	if err != nil {
		t.Fatalf("读取数据集失败: %v", err)
	}

	var dataset test.Dataset
	if err := json.Unmarshal(datasetFile, &dataset); err != nil {
		t.Fatalf("解析数据集失败: %v", err)
	}

	// 遍历数据集进行测试
	for _, tc := range dataset {
		t.Run(tc.ID, func(t *testing.T) {
			fmt.Printf("\n=== 测试用例 %s: %s ===\n", tc.ID, tc.Description)

			// 构建消息列表
			msgs := message.NewMessageList()
			for _, msg := range tc.Input {
				msgs.AddMessage(msg.Role, msg.Content.String())
			}

			// 调用压缩模型
			result, err := cpml.Process(*msgs)
			require.NoError(t, err)

			// 输出结果
			fmt.Printf("\n预期摘要:\n%s\n\n", tc.ExpectedSummary)
			fmt.Printf("实际摘要:\n%s\n\n", result.Content.String())

		})
	}

}
