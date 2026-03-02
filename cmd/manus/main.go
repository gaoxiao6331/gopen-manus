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
	promptFlag := flag.String("prompt", "", "Input prompt for the agent")
	flag.Parse()

	ag := agent.NewManusAgent()

	prompt := strings.TrimSpace(*promptFlag)
	if prompt == "" {
		fmt.Print("Enter your prompt: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		prompt = strings.TrimSpace(line)
	}
	if prompt == "" {
		logger.Warn.Println("Empty prompt provided.")
		return
	}

	logger.Warn.Println("Processing your request...")
	_, err := ag.Run(context.Background(), prompt)
	if err != nil {
		logger.Error.Println(err)
		os.Exit(1)
	}
	logger.Info.Println("Request processing completed.")
}
