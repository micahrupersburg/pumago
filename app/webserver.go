package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

func (app *App) chatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	var req openai.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Query the index
	documents, err := app.queryIndex(req.Messages[0].Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query index: %v", err), http.StatusInternalServerError)
		return
	}
	//allow a mode that doesn't do rag
	var found bool
	req.Messages[0].Content, found = strings.CutPrefix(req.Messages[0].Content, "/pass")
	if !found {
		// Format the prompt
		req.Messages[0].Content = app.formatPrompt(req.Messages[0].Content, documents)
	}

	ctx := context.Background()
	response, err := app.OpenAIClient.CreateChatCompletion(ctx, req)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get response from OpenAI: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (app *App) queryIndex(query string) ([]string, error) {
	// Implement the logic to query the index and return the documents
	// For now, return a mock list of documents
	return []string{"Document 1 content", "Document 2 content"}, nil
}

func (app *App) formatPrompt(userQuery string, documents []string) string {
	prompt := "Here are some documents that might be useful:\n"
	for _, doc := range documents {
		prompt += fmt.Sprintf("<DOC> %s\n", doc)
	}
	prompt += fmt.Sprintf("\nUser query: %s", userQuery)
	return prompt
}
func (ws *WebSockets) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	ws.Connections = append(ws.Connections, conn)
	defer conn.Close()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
	}
}
func (app *App) StartWebServer() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	http.HandleFunc("/v1/chat/completions", app.chatCompletionsHandler)
	http.HandleFunc("/index/ws", app.WebSockets.handleWebSocket)

	go func() {
		log.Println("Starting server on :8080")
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
