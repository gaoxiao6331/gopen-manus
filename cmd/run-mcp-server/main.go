package main

import (
	"flag"
	"os"
	"os/exec"

	"gopen-manus/internal/logger"
)

func main() {
	transport := flag.String("transport", "stdio", "Transport type: stdio")
	command := flag.String("command", "python", "Command to launch MCP server")
	module := flag.String("module", "app.mcp.server", "Python module implementing MCP server")
	flag.Parse()

	_ = transport

	cmd := exec.Command(*command, "-m", *module)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	logger.Info.Println("Starting MCP server...")
	if err := cmd.Run(); err != nil {
		logger.Error.Println(err)
		os.Exit(1)
	}
}
