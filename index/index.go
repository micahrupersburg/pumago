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
)

type Index struct {
	collection string
	embed      chromem.EmbeddingFunc
	port       int
	*chromem.DB
	maxChunkSize int
}

var dbFile = filepath.Join(config.Dir(), "vectors.db")

func (index *Index) Collection() *chromem.Collection {
	return index.GetCollection(index.collection, index.embed)
}
func DefaultIndex() Index {

	db := chromem.NewDB()
	index := Index{
		collection:   "puma-all",
		embed:        chromem.NewEmbeddingFuncOpenAI(os.Getenv("OPENAI_API_KEY"), "text-embedding-3-small"),
		port:         9991,
		DB:           db,
		maxChunkSize: 1024,
	}

	if _, err := os.Stat(dbFile); !os.IsNotExist(err) {
		err = db.ImportFromFile(dbFile, "")
		if err != nil {
			log.Fatalf("Failed to read db file: %v", err)
		}
		if index.Collection() == nil {
			log.Fatalf("Reading index failed due to collection %s not found", index.collection)
		}
	}
	_, err := index.CreateCollection(index.collection, make(map[string]string), index.embed)
	if err != nil {
		log.Fatalf("failed to create index collection %s", index.collection)
	}
	return index
}

// Returns id->content map
func (index *Index) Query(query string, limit int) ([]content.Content, error) {
	ctx := context.Background()
	out := make([]content.Content, 0)
	c := index.GetCollection(index.collection, index.embed)
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

	docRes, err := c.Query(ctx, query, limit, nil, nil)

	for _, res := range docRes {
		out = append(out, docToContent(res.ID, res.Content, res.Metadata))
		//if res.Similarity > 0.7 {
		log.Printf("Document %+v (similarity: %f):", res.Metadata, res.Similarity)
		//}
	}

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
	}
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
	return index.DB.ExportToFile(dbFile, false, "")
}
