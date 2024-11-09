package main

import (
	"flag"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
	"pumago/content"
	"pumago/content/sources"
	"pumago/index"
	"pumago/server"
	"time"
)

func main() {
	flag.Bool("verbose", false, "enable verbose logging")
	flag.Bool("nosource", false, "Don't start sources")
	flag.Bool("rebuild-index", false, "Rebuild Indexes From DB")
	flag.Parse()
	nosource := flag.Lookup("nosource").Value.(flag.Getter).Get().(bool)
	rebuildIndex := flag.Lookup("rebuild-index").Value.(flag.Getter).Get().(bool)
	if rebuildIndex {
		index.Clean()
	}
	appSources := make([]content.Source, 0)

	appSources = append(appSources, sources.SafariBrowser())
	for _, browser := range sources.AllChromeProfiles() {
		appSources = append(appSources, browser)
	}
	appSources = append(appSources, sources.DefaultDrive())
	theIndex := index.DefaultIndex()
	completions := make(map[string]chan content.Content)
	app := App{
		Index:            theIndex,
		Sources:          appSources,
		ScrapeEvery:      5 * time.Minute,
		ContentQueue:     make(chan content.Content, 1000),
		DB:               content.DefaultDB(),
		CompletionQueues: completions,
		WebServer: server.WebServer{
			MyApiKey:     "123456",
			Port:         8888,
			OpenAIClient: openai.NewClient(os.Getenv("OPENAI_API_KEY")),
			Index:        theIndex,
			Outputs:      completions,
		},
	}

	log.Printf("Starting App")
	// Start a worker to process the queue
	log.Printf("Starting Worker")
	go app.ProcessQueue()

	if rebuildIndex {
		log.Printf("Rebuilding Index")
		err := app.DB.UpdateAll(content.NEW)
		if err != nil {
			log.Fatalf("Failed to update all new content: %v", err)
		}
		log.Printf("Loading all new content")
		all, err := app.DB.All(content.NEW)
		if err != nil {
			log.Fatalf("Failed to load all new content: %v", err)
		}
		for _, c := range all {
			app.ContentQueue <- c
		}
		log.Printf("Done Loading all new content %d", len(all))
	}

	go app.Index.StartAutoSaver()
	// Start fetching history every 5 minutes
	log.Printf("Starting Timer for browser scraper")
	if !nosource {
		for _, source := range app.Sources {
			go app.StartSource(source)
		}
	}

	app.WebServer.StartWebServer()

	app.Index.SaveIfDirty() //try to do last save

}
