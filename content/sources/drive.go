package sources

import (
	"context"
	"fmt"
	"io"
	"log"
	"pumago/content"
	"strconv"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Drive struct {
	*drive.Service
	PageSize int64
}

// DefaultDrive initializes the Google Drive service using OAuth2 credentials.
func DefaultDrive() *Drive {
	ctx := context.Background()

	client := getClient(drive.DriveScope)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	return &Drive{PageSize: 10, Service: srv}
}
func (d *Drive) Origin() content.Origin {
	return content.GOOGLE_DRIVE
}
func (d *Drive) FetchContent(state map[string]string) ([]content.Content, error) {
	var lastRead int64 = 0
	stateKey := "last_read"
	stateValue, ok := state[stateKey]
	if ok {
		lastRead, _ = strconv.ParseInt(stateValue, 10, 64)
	}
	now := time.Now().Unix()
	query := "trashed=false"
	if lastRead > 0 {
		query += fmt.Sprintf(" and viewedByMeTime > '%s'", fmt.Sprintf(time.Unix(lastRead, 0).Format(time.RFC3339)))
	}
	r, err := d.Files.List().PageSize(d.PageSize).Fields("files(*)").Q("trashed=false and viewedByMeTime > '2017-06-01T12:00:00'").OrderBy("viewedByMeTime desc, name").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	out := make([]content.Content, 0)
	fmt.Println("Files:")
	for _, i := range r.Files {
		file, err := d.downloadFile(i)
		if err != nil {
			log.Printf("Error downloading file: %v", err)
		} else {
			out = append(out, *file)
		}
	}
	state[stateKey] = fmt.Sprintf("%d", now)
	return out, nil
}

func (d *Drive) downloadFile(file *drive.File) (*content.Content, error) {
	res, err := d.Files.Export(file.Id, "text/plain").Download()
	if err != nil {
		return nil, fmt.Errorf("Unable to download file: %v", err)
	}
	defer res.Body.Close()
	all, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	contentStr := string(all)
	//log.Printf("Downloaded %+v\n", file)
	if file.ViewedByMeTime == "" {
		file.ViewedByMeTime = file.ModifiedTime
	}
	if file.ViewedByMeTime == "" {
		file.ViewedByMeTime = file.CreatedTime
	}
	viewedByMeTime, err := time.Parse(time.RFC3339, file.ViewedByMeTime)
	if err != nil {
		log.Println(err)
		viewedByMeTime = time.Now()
	}
	fmt.Printf("Downloaded %s\n", file.Name)
	entry := content.Content{
		ID:                 file.Id,
		URL:                file.WebViewLink,
		Title:              file.Name,
		LastModifiedMillis: viewedByMeTime.Unix(),
		Origin:             content.GOOGLE_DRIVE,
		Content:            contentStr,
	}

	return &entry, nil
}
