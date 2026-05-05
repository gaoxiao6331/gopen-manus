package tool

// Result mirrors OpenManus ToolResult.
type Result struct {
	Output      string
	Error       string
	Base64Image *string
}

func (r Result) String() string {
	if r.Error != "" {
		return r.Error
	}
	return r.Output
}
