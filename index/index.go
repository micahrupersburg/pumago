package index

import (
	"context"
	"fmt"
	"github.com/philippgille/chromem-go"
	"log"
	"os"
	"path/filepath"
	"pumago/config"
	"pumago/content"
	"strconv"
	"time"
)

type Index struct {
	port         int
	collection   *chromem.Collection
	maxChunkSize int
	db           *chromem.DB
	SaveOnDirty  bool
	Thresh       float32
}

var dirtyCount int
var dbFile = filepath.Join(config.Dir(), "vectors.db")

func Clean() {
	log.Printf("Cleaning index")
	err := os.Remove(dbFile)
	if err != nil {
		log.Printf("Failed to remove db file: %v", err)
	}
}
func DefaultIndex() Index {
	db := chromem.NewDB()
	embed := chromem.NewEmbeddingFuncOpenAI(os.Getenv("OPENAI_API_KEY"), "text-embedding-3-small")
	collectionName := "puma-all"
	var collection *chromem.Collection
	if _, err := os.Stat(dbFile); !os.IsNotExist(err) {
		err = db.ImportFromFile(dbFile, "")
		if err != nil {
			log.Fatalf("Failed to read db file: %v", err)
		}
		collection = db.GetCollection(collectionName, embed)
	}
	if collection == nil {
		var err error
		collection, err = db.CreateCollection(collectionName, make(map[string]string), embed)
		if err != nil {
			log.Fatalf("failed to create index collection %s", collectionName)
		}
	}

	index := Index{
		collection:   collection,
		port:         9991,
		db:           db,
		maxChunkSize: 1024,
		Thresh:       0.2,
	}

	return index
}

// Returns id->content map
func (index *Index) Query(query string, limit int) ([]content.Content, error) {
	ctx := context.Background()
	out := make([]content.Content, 0)
	c := index.collection
	if c == nil {
		log.Printf("collection does not exist")
		return out, nil
	}
	if c.Count() < limit {
		limit = c.Count()
	}
	if limit < 1 {
		log.Printf("no documents in collection or limit is 0")
		return out, nil
	}
	var min float32 = 1.0
	var max float32 = 0.0
	docRes, err := c.Query(ctx, query, limit, nil, nil)

	for _, res := range docRes {

		//if res.Similarity > 0.7 {
		if res.Similarity > index.Thresh {
			out = append(out, docToContent(res.ID, res.Content, res.Metadata))
		}
		if res.Similarity < min {
			min = res.Similarity
		}
		if res.Similarity > max {
			max = res.Similarity
		}
		//}
	}
	log.Printf("Min similarity: %f, Max similarity: %f filtered out %d", min, max, len(docRes)-len(out))

	if err != nil {
		return nil, fmt.Errorf("Failed to query: %v", err)
	}
	return out, nil
}

func (index *Index) splitDoc(doc chromem.Document) []chromem.Document {
	contentLength := len(doc.Content)
	out := make([]chromem.Document, 0)
	for i := 0; i < contentLength; i += index.maxChunkSize {
		end := i + index.maxChunkSize
		if end > contentLength {
			end = contentLength
		}
		chunkID := fmt.Sprintf("%s/%d", doc.ID, i/index.maxChunkSize+1)
		chunk := chromem.Document{ID: chunkID, Content: doc.Content[i:end], Metadata: doc.Metadata}
		out = append(out, chunk)
	}
	return out
}
func (index *Index) doc(doc content.Content) []chromem.Document {
	return index.splitDoc(chromem.Document{
		ID:      doc.ID,
		Content: doc.Content,
		Metadata: map[string]string{
			"Title":              doc.Title,
			"LastModifiedMillis": fmt.Sprintf("%d", doc.LastModifiedMillis),
			"Fragment":           fmt.Sprintf("%d", doc.Fragment),
			"Origin":             doc.Origin.String(),
			"Status":             doc.Status.String(),
			"URL":                doc.URL,
		},
	})
}
func docToContent(id string, docContent string, metadata map[string]string) content.Content {
	lastModified, _ := strconv.ParseInt(metadata["LastModifiedMillis"], 10, 64)
	origin, _ := content.ParseOrigin(metadata["Origin"])
	status, _ := content.ParseStatus(metadata["Status"])
	fragment, _ := strconv.Atoi(metadata["Fragment"])
	return content.Content{
		ID:                 id,
		Content:            docContent,
		Title:              metadata["Title"],
		LastModifiedMillis: lastModified,
		Fragment:           fragment,
		Origin:             origin,
		Status:             status,
		URL:                metadata["URL"],
	}
}
func (index *Index) Add(data content.Content) error {
	ctx := context.Background()
	c := index.collection
	docs := index.doc(data)
	err := c.AddDocuments(ctx, docs, 1)
	if err == nil {
		dirtyCount += len(docs)
		if dirtyCount > 100 {
			err = index.SaveIfDirty()
		}
	}
	return err
}

func (index *Index) SaveIfDirty() error {
	if dirtyCount == 0 || !index.SaveOnDirty {
		return nil
	}
	return index.Save()
}
func (index *Index) Save() error {
	log.Printf("Saving index %d", index.collection.Count())
	err := index.db.ExportToFile(dbFile, false, "", index.collection.Name)
	if err == nil {
		dirtyCount = 0
	}
	return err
}
func (index *Index) StartAutoSaver() {
	index.SaveOnDirty = true
	index.SaveIfDirty() //just in case its already dirty
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := index.SaveIfDirty()
			if err != nil {
				log.Printf("Failed to save index: %v", err)
			}
		}
	}

}
