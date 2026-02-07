package agent

import (
	"fmt"
	"os"

	"memci/context"
)

// BuildSystemPrompts 在 sys segment 中创建模块化的系统提示词 Page
func BuildSystemPrompts(contextMgr *context.ContextManager) error {
	// 使用系统级方法（绕过权限检查）
	sysSeg, err := contextMgr.GetSegmentSystem("sys")
	if err != nil {
		return fmt.Errorf("failed to get sys segment: %w", err)
	}

	rootIndex := sysSeg.GetRootIndex()

	genePagePrompt, err := os.ReadFile("./prompts/gene_page.md")
	if err != nil {
		return fmt.Errorf("failed to read gene page: %w", err)
	}
	contextPrompt, err := os.ReadFile("./prompts/context_page.md")
	if err != nil {
		return fmt.Errorf("failed to read content page: %w", err)
	}
	skillPrompt, err := os.ReadFile("./prompts/context_manage_skill_page.md")
	if err != nil {
		return fmt.Errorf("failed to read skill page: %w", err)
	}
	workflowPrompt, err := os.ReadFile("./prompts/workflow_page.md")
	if err != nil {
		return fmt.Errorf("failed to read workflow page: %w", err)
	}

	genePage, err := contextMgr.CreateDetailPageSystem(
		"Memci Gene",
		"自我认知与禀赋",
		string(genePagePrompt),
		rootIndex,
	)
	if err != nil {
		return fmt.Errorf("failed to create gene page: %w", err)
	}
	if err := contextMgr.ExpandDetailsSystem(genePage); err != nil {
		return fmt.Errorf("failed to expand ATTP protocol page: %w", err)
	}

	contextPage, err := contextMgr.CreateDetailPageSystem(
		"Context System Instroduction",
		"上下文管理系统介绍",
		string(contextPrompt),
		rootIndex,
	)
	if err != nil {
		return fmt.Errorf("failed to create context system page: %w", err)
	}
	if err := contextMgr.ExpandDetailsSystem(contextPage); err != nil {
		return fmt.Errorf("failed to expand context system page: %w", err)
	}
	
	skillPage, err := contextMgr.CreateDetailPageSystem(
		"Memci Skills",
		"上下文管理技能指南",
		string(skillPrompt),
		rootIndex,
	)
	if err != nil {
		return fmt.Errorf("failed to create skill page: %w", err)
	}
	if err := contextMgr.ExpandDetailsSystem(skillPage); err != nil {
		return fmt.Errorf("failed to hide skill page: %w", err)
	}


	workflowPage, err := contextMgr.CreateDetailPageSystem(
		"Memci Workflow",
		"回答每个问题需要遵循的workflow",
		string(workflowPrompt),
		rootIndex,
	)
	if err != nil {
		return fmt.Errorf("failed to create workflow page: %w", err)
	}
	if err := contextMgr.ExpandDetailsSystem(workflowPage); err != nil {
		return fmt.Errorf("failed to hide workflow page: %w", err)
	}
	return nil
}
