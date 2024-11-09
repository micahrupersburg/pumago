package main

import (
	"log"
	"pumago/content"
	"pumago/index"
	"pumago/server"
	"time"
)

type App struct {
	Index            index.Index
	DB               content.DB
	Sources          []content.Source
	ContentQueue     chan content.Content
	ScrapeEvery      time.Duration
	CompletionQueues map[string]chan content.Content
	WebServer        server.WebServer
}

func (app *App) StartSource(source content.Source) {
	app.processSource(source)
	ticker := time.NewTicker(app.ScrapeEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.processSource(source)
		}
	}
}

func (app *App) processSource(source content.Source) {
	log.Printf("Fetching content from %s", source.Origin())
	settings, err := app.DB.LoadSettings(source.Origin().String())
	if err != nil {
		log.Printf("Failed to load settings: %v", err)
		return
	}
	contents, err := source.FetchContent(settings)
	if err != nil {
		log.Printf("Failed to fetch contents: %v from source %s", err, source.Origin())
		return
	}
	log.Printf("Fetched %d contents from source %s", len(contents), source.Origin())
	app.DB.SaveSettings(source.Origin().String(), settings)
	for _, data := range contents {
		data = data.Shrink()
		app.ContentQueue <- data
	}
}

func (app *App) ProcessQueue() {
	for data := range app.ContentQueue {
		err := app.DB.Add(data)
		if err != nil {
			log.Printf("Didn't add content to database: %v", err)
			continue
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
		for _, queue := range app.CompletionQueues {
			queue <- data
		}
		log.Printf("Added content to %d queues", len(app.CompletionQueues))
	}

}
