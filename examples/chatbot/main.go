package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	openai "github.com/jaylidev/openai-go-sdk"
	"go.uber.org/zap"
)

func main() {
	client := openai.NewClient(
		openai.WithModel(openai.DeepSeekV4Flash),
		openai.WithAPIKey(os.Getenv("DEEPSEEK_API_KEY")),
		openai.WithLogger(zap.NewExample()),
		openai.WithLogLevel(openai.LogLevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🤖 DeepSeek Chatbot (输入 /exit 退出，Ctrl+C 退出)")
	fmt.Println(strings.Repeat("─", 50))

	var history []openai.Message

	inputCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		close(inputCh)
	}()

	for {
		fmt.Print("\n你: ")

		var input string
		select {
		case <-sigCh:
			fmt.Println("\n👋 再见!")
			return
		case <-ctx.Done():
			return
		case line, ok := <-inputCh:
			if !ok {
				return
			}
			input = strings.TrimSpace(line)
		}

		if input == "" {
			continue
		}
		if input == "/exit" {
			fmt.Println("👋 再见!")
			return
		}

		history = append(history, openai.UserMessage(input))

		stream, err := client.Chat().
			SystemPrompt("你是中文AI助手").
			Messages(history).
			Temperature(0.7).
			Stream(ctx)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		fmt.Print("AI: ")
		var fullContent string
		for stream.Next() {
			delta := stream.Delta()
			fmt.Print(delta.Content)
			fullContent += delta.Content
		}
		fmt.Println()
		stream.Close()

		if fullContent != "" {
			history = append(history, openai.AssistantMessage(fullContent))
		}
	}
}
