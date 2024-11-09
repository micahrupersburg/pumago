package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pumago/content"
	"pumago/index"
	"time"
)

type WebServer struct {
	Port         int
	OpenAIClient *openai.Client
	MyApiKey     string
	Index        index.Index
	Outputs      map[string]chan content.Content
}
type Handler func(w http.ResponseWriter, req openai.ChatCompletionRequest)

func writeStreamResponse(w http.ResponseWriter, response openai.ChatCompletionStreamResponse) {
	marshal, err := json.Marshal(response)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		return
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", marshal)
	if err != nil {
		log.Printf("Error writing response: %v\n", err)
		return
	}
	w.(http.Flusher).Flush()
}

func (ws *WebServer) chatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
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
	input := req.Messages[len(req.Messages)-1].Content

	cmd, input := command(input)
	log.Printf("parse command '%s', '%s'", cmd.String(), input)
	req.Messages[len(req.Messages)-1].Content = input
	handler := ws.chatDefaultStreamHandler

	switch cmd {
	case Watch:
		handler = ws.handleWatchCommand
	case Query:
		handler = ws.handleQueryCommand
	case Raw:
		handler = ws.chatDefaultStreamHandler
	default:
		prompt, err := ws.RagPrompt(input)
		if err != nil {
			estr := fmt.Sprintf("Rag Failure %+v", err)
			http.Error(w, estr, http.StatusBadRequest)
			return
		}
		log.Printf("Prompt used to send to OpenAI %s", prompt)
		req.Messages[len(req.Messages)-1].Content = prompt
		handler = ws.chatDefaultStreamHandler
	}

	handler(w, req)
}

func (ws *WebServer) StartWebServer() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", ws.Port),
		Handler: nil,
	}

	http.HandleFunc("/v1/chat/completions", ws.chatCompletionsHandler)

	go func() {
		log.Printf("Starting server on :%d", ws.Port)
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
