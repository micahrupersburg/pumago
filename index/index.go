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
)

type Index struct {
	collection string
	embed      chromem.EmbeddingFunc
	port       int
	*chromem.DB
	maxChunkSize int
}

var dbFile = filepath.Join(config.Dir(), "vectors.db")

func DefaultIndex() Index {

	db := chromem.NewDB()
	if _, err := os.Stat(dbFile); !os.IsNotExist(err) {
		err = db.ImportFromFile(dbFile, "")
		if err != nil {
			log.Fatalf("Failed to read db file: %v", err)
		}
	}

	return Index{
		collection:   "openapi",
		embed:        chromem.NewEmbeddingFuncOpenAICompat("http://localhost:9991", "", "", nil),
		port:         9991,
		DB:           db,
		maxChunkSize: 1024,
	}
}

func (index *Index) Query(query string, limit int) {
	ctx := context.Background()
	c := index.GetCollection(index.collection, index.embed)
	docRes, err := c.Query(ctx, query, limit, nil, nil)
	// Here you could filter out any documents whose similarity is below a certain threshold.
	// if docRes[...].Similarity < 0.5 { ...

	for _, res := range docRes {
		//if res.Similarity > 0.7 {
		log.Printf("Document %d (similarity: %f):", res.ID, res.Similarity)
		//}
	}

	if err != nil {
		log.Fatalf("Failed to query db: %v", err)
		return
	}
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
		chunk := chromem.Document{ID: chunkID, Content: doc.Content[i:end]}
		out = append(out, chunk)
	}
	return out
}
func (index *Index) doc(doc content.Content) []chromem.Document {
	return index.splitDoc(chromem.Document{
		ID:      doc.ID,
		Content: doc.Content,
	})
}

func (index *Index) Add(data content.Content) error {
	ctx := context.Background()
	c, err := index.GetOrCreateCollection(index.collection, nil, index.embed)
	if err != nil {
		log.Fatalf("Failed to get collection: %v", err)
	}
	return c.AddDocuments(ctx, index.doc(data), 1)
}
func (index *Index) Save() error {
	return index.DB.ExportToFile(dbFile, false, "", index.collection)
}
