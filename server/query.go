package server

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"pumago/content"
)

func (ws *WebServer) queryIndex(query string) ([]content.Content, error) {
	return ws.Index.Query(query, 10)
}

func (ws *WebServer) RagPrompt(prompt string) (string, error) {
	// Query the index
	documents, err := ws.queryIndex(prompt)
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
	prompt += fmt.Sprintf("\n\n<Prompt>%s", userQuery)
	return prompt
}
func (ws *WebServer) handleQueryCommand(w http.ResponseWriter, req openai.ChatCompletionRequest) {

	docs, err := ws.queryIndex(req.Messages[len(req.Messages)-1].Content)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	for i, doc := range docs {
		//log.Printf("Sending doc %s", doc.Markdown())
		response := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Index: i,
					Delta: openai.ChatCompletionStreamChoiceDelta{
						Role:    openai.ChatMessageRoleAssistant,
						Content: doc.Markdown(),
					},
				},
			},
			Model: "vectors",
			ID:    "stream-response-id:" + doc.ID,
		}
		writeStreamResponse(w, response)
		if i == len(docs)-1 {
			w.Write([]byte("data: [DONE]\n\n"))
		}
	}
}

func (ws *WebServer) handleWatchCommand(w http.ResponseWriter, req openai.ChatCompletionRequest) {
	key := uuid.New().String()
	channel := make(chan content.Content)
	ws.Outputs[key] = channel
	defer func() {
		delete(ws.Outputs, key)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	i := 0
	for doc := range channel {
		log.Printf("Sending doc %s", doc.ID)
		response := openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Index: i,
					Delta: openai.ChatCompletionStreamChoiceDelta{
						Role:    openai.ChatMessageRoleAssistant,
						Content: doc.Markdown(),
					},
				},
			},
			Model: "vectors",
			ID:    "stream-response-id:" + doc.ID,
		}
		writeStreamResponse(w, response)
		i++
	}

}
