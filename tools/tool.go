package tools

type H = map[string]interface{}

const (
	ToolTypeFunction = "function"
)

// FunctionDefinition represents a function definition for a tool
type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
	Parameters  H      `json:"parameters"`
}

// Tool represents a tool that can be called by the model
type Tool struct {
	Type     string              `json:"type"`
	Function *FunctionDefinition `json:"function,omitempty"`
}

type Field struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}

type ParameterBuilder struct {
	fields H
	required []string
}

func NewParameterBuilder() *ParameterBuilder {
	return &ParameterBuilder{
		fields: make(H, 0),
		required: make([]string, 0),
	}
}

func (b *ParameterBuilder) AddField(field Field)  *ParameterBuilder {
	b.fields[field.Name] = H{
		"type": field.Type,
		"description": field.Description,
	}
	if field.Required {
		b.required = append(b.required, field.Name)
	}
	return b
}

func (b *ParameterBuilder) Build() H{
	return H{
		"type": "object",
		"properties": b.fields,
		"required": b.required,
	}
}

type FunctionTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
	Parameters  H    `json:"parameters"`
}



type ToolList struct {
	Tools []FunctionTool
}

func NewToolList() *ToolList {
	return &ToolList{
		Tools: make([]FunctionTool, 0),
	}
}

func (tl *ToolList) ConvertToOaiFormat() []Tool {
	var tools []Tool
	for _, tool := range tl.Tools {
		tools = append(tools, Tool{
			Type: ToolTypeFunction,
			Function: &FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Strict:      tool.Strict,
				Parameters:  tool.Parameters,
			},
		})
	}
	return tools
}