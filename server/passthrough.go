package server

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"net/http"
)

func (ws *WebServer) chatDefaultStreamHandler(w http.ResponseWriter, req openai.ChatCompletionRequest) {
	ctx := context.Background()
	stream, err := ws.OpenAIClient.CreateChatCompletionStream(ctx, req)
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
		writeStreamResponse(w, response)
	}
}
