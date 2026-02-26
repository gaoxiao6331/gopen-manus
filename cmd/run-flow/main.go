package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gopen-manus/internal/agent"
	"gopen-manus/internal/flow"
	"gopen-manus/internal/logger"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your prompt: ")
	prompt, _ := reader.ReadString('\n')
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		logger.Warn.Println("Empty prompt provided.")
		return
	}

	agents := map[string]agent.Agent{
		"manus": agent.NewToolCallAgent("manus"),
	}

	factory := &flow.Factory{}
	flowExec, err := factory.Create(flow.FlowTypePlanning, agents, map[string]any{})
	if err != nil {
		logger.Error.Println(err)
		os.Exit(1)
	}

	logger.Warn.Println("Processing your request...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	start := time.Now()
	result, err := flowExec.Execute(ctx, prompt)
	if err != nil {
		logger.Error.Println(err)
		os.Exit(1)
	}
	elapsed := time.Since(start)
	logger.Info.Printf("Request processed in %.2f seconds", elapsed.Seconds())
	if result != "" {
		logger.Info.Println(result)
	}
}
