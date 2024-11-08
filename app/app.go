package app

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"log"
	"pumago/content"
	"pumago/index"
	"time"
)

type App struct {
	Index        index.Index
	DB           content.DB
	Sources      []content.Source
	ContentQueue chan content.Content
	ScrapeEvery  time.Duration
	OpenAIClient *openai.Client
	WebSockets   WebSockets
}

type WebSockets struct {
	Upgrader    websocket.Upgrader
	Connections []*websocket.Conn
}

func (app *App) Run() {
	log.Printf("Starting Index")
	////err := app.Index.Launch()
	////if err != nil {
	////	log.Fatalf("Failed to launch index: %v", err)
	//}

	// Start a worker to process the queue
	log.Printf("Starting Worker")
	go app.processQueue()

	// Start fetching history every 5 minutes
	log.Printf("Starting Timer for browser scraper")
	go app.startScraping()
	app.StartWebServer()
}
func (app *App) scrape() {
	log.Println("Fetching content from sources")
	var dirty = false
	for _, source := range app.Sources {
		contents, err := source.FetchContent()
		if err != nil {
			log.Printf("Failed to fetch contents: %v from source %v", err, source)
			return
		}
		log.Printf("Fetched %d contents from source %v", len(contents), source)
		for _, data := range contents {
			dirty = true
			app.ContentQueue <- data
		}
	}
	if dirty {
		err := app.Index.Save()
		if err != nil {
			log.Printf("Failed to save index: %v", err)
		}
	}
}
func (app *App) startScraping() {
	app.scrape()
	ticker := time.NewTicker(app.ScrapeEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.scrape()
		}
	}
}

func (app *App) processQueue() {
	for data := range app.ContentQueue {
		err := app.DB.Add(data)
		if err != nil {
			log.Printf("Failed to add content to database: %v", err)
			continue
		}
		err = app.SendDocsToWebSockets(data)
		if err != nil {
			log.Printf("Failed to add content to web sockets: %v", err)
		}
		err = app.Index.Add(data)
		if err != nil {
			log.Printf("Failed to add content to index: %v", err)
			err := app.DB.Failed(data)
			if err != nil {
				log.Printf("Failed to add update database: %v", err)
			}
		}
		err = app.DB.Processed(data)
		if err != nil {
			log.Printf("Failed to add update database: %v", err)
		}
	}

}

type WebSocketEvent struct {
	Created content.Content   `json:"data"`
	Matches []content.Content `json:"matches"`
}

func (app *App) SendDocsToWebSockets(data content.Content) error {
	if data.Origin != content.AUDIO && data.Origin != content.CHAT {
		return nil
	}
	responseData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	for _, conn := range app.WebSockets.Connections {
		err := conn.WriteMessage(websocket.TextMessage, responseData)
		if err != nil {
			log.Printf("Failed to write message: %v", err)
		}
	}
	return nil
}
