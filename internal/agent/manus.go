package agent

type ManusAgent struct {
	*ToolCallAgent
}

func NewManusAgent() *ManusAgent {
	return &ManusAgent{
		ToolCallAgent: NewToolCallAgent("manus"),
	}
}
