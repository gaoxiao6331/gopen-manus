package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopen-manus/internal/agent"
	"gopen-manus/internal/logger"
)

func main() {
	connection := flag.String("connection", "stdio", "Connection type: stdio or sse")
	serverURL := flag.String("server-url", "http://127.0.0.1:8000/sse", "URL for SSE connection")
	interactive := flag.Bool("interactive", false, "Run in interactive mode")
	prompt := flag.String("prompt", "", "Single prompt to execute and exit")
	command := flag.String("command", "python", "Command for stdio connection")
	module := flag.String("module", "app.mcp.server", "Python module for stdio connection")
	flag.Parse()

	ag := agent.NewMCPAgent()
	ctx := context.Background()

	args := []string{"-m", *module}
	if err := ag.Initialize(ctx, *connection, *serverURL, *command, args); err != nil {
		logger.Error.Println(err)
		os.Exit(1)
	}
	defer ag.Cleanup(ctx)

	if *prompt != "" {
		_, _ = ag.Run(ctx, *prompt)
		return
	}
	if *interactive {
		fmt.Println("\nMCP Agent Interactive Mode (type 'exit' to quit)\n")
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("\nEnter your request: ")
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "exit" || line == "quit" || line == "q" {
				break
			}
			response, err := ag.Run(ctx, line)
			if err != nil {
				logger.Error.Println(err)
				continue
			}
			fmt.Printf("\nAgent: %s\n", response)
		}
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your prompt: ")
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		logger.Warn.Println("Empty prompt provided.")
		return
	}
	logger.Warn.Println("Processing your request...")
	_, _ = ag.Run(ctx, line)
	logger.Info.Println("Request processing completed.")
}
