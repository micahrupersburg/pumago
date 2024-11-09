package content

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func AllChromeProfiles() []*Browser {
	profilePaths, err := listHistoryFiles()
	if err != nil {
		return nil
	}

	var browsers []*Browser
	for _, path := range profilePaths {
		browsers = append(browsers, ChromeBrowser(path))
	}
	return browsers
}

func ChromeBrowser(historyPath string) *Browser {

	return &Browser{
		historyPath: historyPath,
		//maxHistory:  100,
		query: `
SELECT 
    COALESCE(title, '') AS title, 
    url, 
    last_visit_time
FROM 
    urls
WHERE 
    last_visit_time > ?
    AND last_visit_time = (
        SELECT MAX(last_visit_time)
        FROM urls AS u
        WHERE u.url = urls.url
    )
ORDER BY 
    last_visit_time DESC
LIMIT 100;`,
		origin: CHROME,
	}
}

// getChromeProfilePath returns the Chrome profile path based on the OS.
func getChromeProfilePath() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome")
	case "linux":
		return filepath.Join(os.Getenv("HOME"), ".config", "google-chrome")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
	default:
		return ""
	}
}

// isChromeProfile checks if a directory contains typical Chrome profile files.
func isChromeProfile(path string) bool {
	requiredFiles := []string{"Preferences", "History", "Bookmarks"}
	for _, file := range requiredFiles {
		if _, err := os.Stat(filepath.Join(path, file)); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func listHistoryFiles() ([]string, error) {
	profilePath := getChromeProfilePath()
	if profilePath == "" {
		return nil, fmt.Errorf("unsupported OS")
	}

	// List all items in the Chrome User Data directory
	files, err := os.ReadDir(profilePath)
	if err != nil {
		return nil, err
	}

	var historyFiles []string
	for _, file := range files {
		if file.IsDir() {
			fullPath := filepath.Join(profilePath, file.Name())
			// Check if the directory has Chrome profile files
			if isChromeProfile(fullPath) {
				historyFiles = append(historyFiles, filepath.Join(fullPath, "History"))
			}
		}
	}

	return historyFiles, nil
}
