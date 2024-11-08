package content

import (
	"regexp"
	"strings"
)

type Origin int

type Status int

const (
	NEW Status = iota
	PROCESSED
	FAILED
)
const (
	CHROME Origin = iota
	SAFARI
	GOOGLE_DRIVE
	AUDIO
	CHAT
)

func (s Status) String() string {
	return [...]string{"NEW", "PROCESSED", "FAILED"}[s]
}
func (s Origin) String() string {
	return [...]string{"CHROME", "SAFARI", "AUDIO", "CHAT"}[s]
}

type Source interface {
	FetchContent() ([]Content, error)
}

// Content represents a single entry in the file history.
type Content struct {
	Origin             Origin `json:"origin"`
	ID                 string `json:"id"`
	URL                string `json:"url"`
	Title              string `json:"title"`
	LastModifiedMillis int64  `json:"last_modified_millis"`
	Fragment           int    `json:"fragment"`
	Content            string `json:"content"`
	Status             Status `json:"status"`
}

// ConvertURLToFilename converts a URL to a Unix-compatible filename.
func ConvertURLToFilename(url string) string {
	// Remove the protocol prefix (e.g., "https://").
	cleanURL := strings.TrimPrefix(url, "http://")
	cleanURL = strings.TrimPrefix(cleanURL, "https://")

	// Replace non-alphanumeric characters with underscores.
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	filename := re.ReplaceAllString(cleanURL, "_")

	// Trim any trailing underscores.
	filename = strings.Trim(filename, "_")

	return filename
}

type ContentManger interface {
	Add(content Content) error
	Processed(content Content) error
	Failed(content Content) error
	Get(origin Origin, id string) (Content, error)
	List(origin Origin, status Status) ([]Content, error)
}
