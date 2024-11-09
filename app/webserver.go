package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pumago/content"
	"regexp"
	"time"
)

type Handler func(w http.ResponseWriter, req openai.ChatCompletionRequest)

type Command int

const (
	None Command = iota
	Raw  Command = iota
	Watch
	Query
)

func (c Command) String() string {
	return [...]string{"raw", "watch", "query"}[c]
}
func ParseCommand(input string) (Command, error) {
	switch input {
	case "raw":
		return Raw, nil
	case "watch":
		return Watch, nil
	case "query":
		return Query, nil
	default:
		return -1, fmt.Errorf("invalid command: %s", input)
	}
}

var CommandRegex = regexp.MustCompile(`^/(\w+)\s*(.*)$`)

func command(input string) (Command, string) {
	matches := CommandRegex.FindStringSubmatch(input)
	if matches == nil {
		return None, input
	}
	parseCommand, err := ParseCommand(matches[1])
	if err != nil {
		return None, input
	}
	return parseCommand, matches[2]
}
func (app *App) handleWatchCommand(w http.ResponseWriter, req openai.ChatCompletionRequest) {

}
func (app *App) handleQueryCommand(w http.ResponseWriter, req openai.ChatCompletionRequest) {
	docs, err := app.queryIndex(req.Messages[0].Content)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	for i, doc := range docs {

		response := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Index: i,
					Delta: openai.ChatCompletionStreamChoiceDelta{
						Role:    openai.ChatMessageRoleAssistant,
						Content: doc.Content,
					},
				},
			},
			Model: "vectors",
			ID:    "stream-response-id:" + doc.ID,
		}
		if i == len(docs)-1 {
			response.Choices[0].FinishReason = "stop"
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, fmt.Sprintf("Failed to receive stream response: %v", err), http.StatusInternalServerError)
			return
		}
		if err := encoder.Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func (app *App) chatDefaultStreamHandler(w http.ResponseWriter, req openai.ChatCompletionRequest) {

	ctx := context.Background()
	stream, err := app.OpenAIClient.CreateChatCompletionStream(ctx, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get response from OpenAI: %v", err), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				w.Write([]byte("data: [DONE]\n\n"))
				break
			}
			http.Error(w, fmt.Sprintf("Failed to receive stream response: %v", err), http.StatusInternalServerError)
			return
		}

		marshal, err := json.Marshal(response)

		if err != nil {
			log.Printf("Error marshalling response: %v\n", err)
			continue
		}
		log.Printf("Send Response %s", response.Choices[0].Delta.Content)
		_, err = fmt.Fprintf(w, "data: %s\n\n", marshal)
		if err != nil {
			log.Printf("Error writing response: %v\n", err)
			return
		}
		w.(http.Flusher).Flush()
	}
}

func (app *App) chatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	var req openai.ChatCompletionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("Request %+v", req)
	if !req.Stream {
		http.Error(w, "Only Streaming Supported", http.StatusBadRequest)
		return
	}
	input := req.Messages[0].Content
	cmd, input := command(input)
	req.Messages[0].Content = input
	handler := app.chatDefaultStreamHandler

	switch cmd {
	case Watch:
		handler = app.handleWatchCommand
	case Query:
		handler = app.handleQueryCommand
	case Raw:
		handler = app.chatDefaultStreamHandler
	default:
		prompt, err := app.RagPrompt(req.Messages[0].Content)
		if err != nil {
			estr := fmt.Sprintf("Rag Failure %+v", err)
			http.Error(w, estr, http.StatusBadRequest)
			return
		}
		log.Printf("Prompt used to send to OpenAI %s", prompt)
		req.Messages[0].Content = prompt
		handler = app.chatDefaultStreamHandler
	}

	handler(w, req)
}

func (app *App) queryIndex(query string) ([]content.Content, error) {
	return app.Index.Query(query, 0)
}

func (app *App) RagPrompt(prompt string) (string, error) {
	// Query the index
	documents, err := app.queryIndex(prompt)
	if err != nil {
		return "", err
	}
	return formatPrompt(prompt, documents), nil
}
func formatPrompt(userQuery string, documents []content.Content) string {
	prompt := "Here are some documents that might be useful:\n"
	for _, doc := range documents {
		prompt += fmt.Sprintf("<DOC> %s\n", doc.Content)
	}
	prompt += fmt.Sprintf("\nUser query: %s", userQuery)
	return prompt
}
func (app *App) StartWebServer() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.WebServerPort),
		Handler: nil,
	}

	http.HandleFunc("/v1/chat/completions", app.chatCompletionsHandler)

	go func() {
		log.Printf("Starting server on :%d", app.WebServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
