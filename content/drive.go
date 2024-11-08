package content

import (
	"context"
	"fmt"
	"io"
	"log"
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

func (d *Drive) FetchContent() ([]Content, error) {
	r, err := d.Files.List().PageSize(d.PageSize).Fields("files(*)").Q("trashed=false").OrderBy("viewedByMeTime desc, name").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	out := make([]Content, 0)
	fmt.Println("Files:")
	for _, i := range r.Files {
		file, err := d.downloadFile(i)
		if err != nil {
			log.Printf("Error downloading file: %v", err)
		} else {
			out = append(out, *file)
		}
	}
	return out, nil
}

func (d *Drive) downloadFile(file *drive.File) (*Content, error) {
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
	entry := Content{
		ID:                 file.Id,
		URL:                file.WebViewLink,
		Title:              file.Name,
		LastModifiedMillis: viewedByMeTime.Unix(),
		Origin:             GOOGLE_DRIVE,
		Content:            contentStr,
	}

	return &entry, nil
}
