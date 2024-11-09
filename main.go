package main

import (
	"flag"
	"github.com/sashabaranov/go-openai"
	"os"
	"pumago/app"
	"pumago/content"
	"pumago/index"
	"time"
)

func main() {
	flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	sources := make([]content.Source, 0)
	//[]content.Source{content.DefaultDrive()}, // , content.SafariBrowser(), content.ChromeBrowser()},
	//sources = append(sources, content.SafariBrowser())
	//for _, browser := range content.AllChromeProfiles() {
	//	sources = append(sources, browser)
	//}
	sources = append(sources, content.DefaultDrive())

	app := app.App{
		Index:         index.DefaultIndex(),
		Sources:       sources,
		ScrapeEvery:   5 * time.Minute,
		ContentQueue:  make(chan content.Content, 1000),
		OpenAIClient:  openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		DB:            content.DefaultDB(),
		MyApiKey:      "123456",
		WebServerPort: 8888,
	}

	app.Run()

}
