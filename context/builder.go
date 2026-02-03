package context

import (
	"fmt"
	"memci/config"
	"memci/llm"
	"memci/logger"
	"memci/message"
	"os"
)

// ContextBuilder 上下文系统构建器
type ContextBuilder struct {
	cfg          *config.Config
	compactModel *llm.CompactModel
	logger       logger.Logger
}

// NewContextBuilder 创建构建器
func NewContextBuilder(cfg *config.Config, compactModel *llm.CompactModel, logger logger.Logger) *ContextBuilder {
	return &ContextBuilder{
		cfg:          cfg,
		compactModel: compactModel,
		logger:       logger,
	}
}

// Build 构建完整的上下文系统
func (cb *ContextBuilder) Build(sysMsgs []message.Message) (*ContextSystem, error) {
	// 1. 创建存储
	storage, err := cb.createStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// 2. 从配置获取参数
	activeSize := cb.cfg.Context.ActiveMaxPages
	archiveMaxToken := cb.cfg.Context.ArchiveMaxToken

	// 3. 创建上下文系统
	ctxSystem := NewContextSystem(
		sysMsgs,
		activeSize,
		archiveMaxToken,
		storage,
		cb.compactModel,
	)

	// 4. 配置 ArchiveChapter 的卸载策略
	if commonSeg := ctxSystem.GetCommonSegment(); commonSeg != nil {
		if archiveChapter := commonSeg.GetArchiveChapter(); archiveChapter != nil {
			unloadConfig := UnloadConfig{
				MinRecallTurns:   cb.cfg.Context.UnloadMinRecallTurns,
				StaleRecallTurns: cb.cfg.Context.UnloadStaleRecallTurns,
				MaxTokenRatio:    cb.cfg.Context.UnloadMaxTokenRatio,
			}
			archiveChapter.SetUnloadConfig(unloadConfig)
		}
	}

	cb.logger.Info("Context system built successfully",
		logger.Int("active_max_pages", activeSize),
		logger.Int("archive_max_token", archiveMaxToken),
		logger.String("storage_dir", cb.cfg.Context.StorageBaseDir))

	return ctxSystem, nil
}

// createStorage 创建 Page Storage
func (cb *ContextBuilder) createStorage() (PageStorage, error) {
	// 确保存储目录存在
	err := os.MkdirAll(cb.cfg.Context.StorageBaseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 根据配置创建存储
	if cb.cfg.Context.StorageUseGzip {
		cb.logger.Info("Using gzip compressed file storage")
		return NewFilePageStorage(cb.cfg.Context.StorageBaseDir, true)
	}

	cb.logger.Info("Using plain JSON file storage")
	return NewFilePageStorage(cb.cfg.Context.StorageBaseDir, false)
}

// BuildWithATTP 构建上下文系统并配置 ATTP 召回工具
func (cb *ContextBuilder) BuildWithATTP(sysMsgs []message.Message, attpTools *ATTPTools) (*ContextSystem, error) {
	ctxSystem, err := cb.Build(sysMsgs)
	if err != nil {
		return nil, err
	}

	// 设置 ATTP 召回工具定义到系统提示词
	toolsDef := attpTools.GetToolDefinitions()
	ctxSystem.SetExtraTools(toolsDef)

	cb.logger.Info("Context system built with ATTP tools")
	return ctxSystem, nil
}

// NewContextBuilderWithDefaults 使用默认配置创建构建器
func NewContextBuilderWithDefaults(cfg *config.Config, logger logger.Logger) *ContextBuilder {
	// 创建压缩模型
	compactModel := llm.NewCompactModel(cfg, logger)

	return NewContextBuilder(cfg, compactModel, logger)
}

// BuildSimple 简化版构建函数，使用默认配置
func BuildSimple(sysPrompt string, logger logger.Logger) (*ContextSystem, error) {
	// 创建默认配置
	cfg := &config.Config{
		Context: config.ContextConfig{
			ActiveMaxPages:       5,
			ArchiveMaxToken:      2000,
			UnloadMinRecallTurns: 10,
			UnloadStaleRecallTurns: 20,
			UnloadMaxTokenRatio:  0.9,
			StorageBaseDir:       "./data/pages",
			StorageUseGzip:       true,
		},
	}

	// 创建构建器
	builder := NewContextBuilderWithDefaults(cfg, logger)

	// 创建系统消息
	sysMsgs := []message.Message{}
	if sysPrompt != "" {
		sysMsgs = append(sysMsgs, message.Message{
			Role:    message.System,
			Content: message.NewContentString(sysPrompt),
		})
	}

	// 构建上下文系统
	return builder.Build(sysMsgs)
}

// ContextSystemBuilder 上下文系统的流式构建器
type ContextSystemBuilder struct {
	cfg           *config.Config
	compactModel  *llm.CompactModel
	logger        logger.Logger
	sysMsgs       []message.Message
	attpTools     *ATTPTools
	storage       PageStorage
	activeSize    int
	archiveMaxToken int
}

// NewContextSystemBuilder 创建流式构建器
func NewContextSystemBuilder() *ContextSystemBuilder {
	return &ContextSystemBuilder{
		sysMsgs:        make([]message.Message, 0),
		activeSize:     5,              // 默认值
		archiveMaxToken: 2000,          // 默认值
	}
}

// WithConfig 设置配置
func (csb *ContextSystemBuilder) WithConfig(cfg *config.Config) *ContextSystemBuilder {
	csb.cfg = cfg
	// 从配置更新参数
	csb.activeSize = cfg.Context.ActiveMaxPages
	csb.archiveMaxToken = cfg.Context.ArchiveMaxToken
	return csb
}

// WithLogger 设置日志
func (csb *ContextSystemBuilder) WithLogger(logger logger.Logger) *ContextSystemBuilder {
	csb.logger = logger
	return csb
}

// WithCompactModel 设置压缩模型
func (csb *ContextSystemBuilder) WithCompactModel(model *llm.CompactModel) *ContextSystemBuilder {
	csb.compactModel = model
	return csb
}

// WithSystemPrompt 设置系统提示词
func (csb *ContextSystemBuilder) WithSystemPrompt(prompt string) *ContextSystemBuilder {
	csb.sysMsgs = append(csb.sysMsgs, message.Message{
		Role:    message.System,
		Content: message.NewContentString(prompt),
	})
	return csb
}

// WithSystemMessage 添加系统消息
func (csb *ContextSystemBuilder) WithSystemMessage(msg message.Message) *ContextSystemBuilder {
	csb.sysMsgs = append(csb.sysMsgs, msg)
	return csb
}

// WithATTPTools 设置 ATTP 工具
func (csb *ContextSystemBuilder) WithATTPTools(attpTools *ATTPTools) *ContextSystemBuilder {
	csb.attpTools = attpTools
	return csb
}

// WithStorage 设置存储
func (csb *ContextSystemBuilder) WithStorage(storage PageStorage) *ContextSystemBuilder {
	csb.storage = storage
	return csb
}

// WithActiveSize 设置 ActiveChapter 大小
func (csb *ContextSystemBuilder) WithActiveSize(size int) *ContextSystemBuilder {
	csb.activeSize = size
	return csb
}

// WithArchiveMaxToken 设置 ArchiveChapter 最大 token
func (csb *ContextSystemBuilder) WithArchiveMaxToken(maxToken int) *ContextSystemBuilder {
	csb.archiveMaxToken = maxToken
	return csb
}

// Build 构建上下文系统
func (csb *ContextSystemBuilder) Build() (*ContextSystem, error) {
	// 使用提供的存储或创建默认存储
	storage := csb.storage
	if storage == nil && csb.cfg != nil {
		var err error
		storage, err = csb.createDefaultStorage()
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}
	}

	if storage == nil {
		// 使用内存存储作为后备
		storage = NewMemoryPageStorage()
	}

	// 创建上下文系统
	ctxSystem := NewContextSystem(
		csb.sysMsgs,
		csb.activeSize,
		csb.archiveMaxToken,
		storage,
		csb.compactModel,
	)

	// 配置卸载策略
	if csb.cfg != nil {
		if commonSeg := ctxSystem.GetCommonSegment(); commonSeg != nil {
			if archiveChapter := commonSeg.GetArchiveChapter(); archiveChapter != nil {
				unloadConfig := UnloadConfig{
					MinRecallTurns:   csb.cfg.Context.UnloadMinRecallTurns,
					StaleRecallTurns: csb.cfg.Context.UnloadStaleRecallTurns,
					MaxTokenRatio:    csb.cfg.Context.UnloadMaxTokenRatio,
				}
				archiveChapter.SetUnloadConfig(unloadConfig)
			}
		}
	}

	// 配置 ATTP 工具
	if csb.attpTools != nil {
		toolsDef := csb.attpTools.GetToolDefinitions()
		ctxSystem.SetExtraTools(toolsDef)
	}

	return ctxSystem, nil
}

// createDefaultStorage 创建默认存储
func (csb *ContextSystemBuilder) createDefaultStorage() (PageStorage, error) {
	if csb.cfg == nil {
		return NewMemoryPageStorage(), nil
	}

	storageDir := csb.cfg.Context.StorageBaseDir
	if storageDir == "" {
		storageDir = "./data/pages"
	}

	useGzip := csb.cfg.Context.StorageUseGzip

	return NewFilePageStorage(storageDir, useGzip)
}
