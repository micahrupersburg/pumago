package main

import (
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"net/http"
	"os"
	"pumago/app"
	"pumago/content"
	"pumago/index"
	"time"
)

func main() {

	app := app.App{
		Index:        index.DefaultIndex(),
		Sources:      []content.Source{content.DefaultDrive()}, // , content.SafariBrowser(), content.ChromeBrowser()},
		ScrapeEvery:  5 * time.Minute,
		ContentQueue: make(chan content.Content, 1000),
		OpenAIClient: openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		DB:           content.DefaultDB(),
		WebSockets: app.WebSockets{
			Upgrader: websocket.Upgrader{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
			Connections: make([]*websocket.Conn, 0),
		},
	}

	app.Run()

}
