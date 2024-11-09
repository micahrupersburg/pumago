package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"pumago/config"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(scope ...string) *http.Client {

	// Path to the credentials.json and token.json files
	credFile := filepath.Join(config.BinDir(), "credentials.json")

	b, err := os.ReadFile(credFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	configJson, err := google.ConfigFromJSON(b, scope...)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := filepath.Join(config.Dir(), "token.json")
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(configJson)
		saveToken(tokFile, tok)
	}
	return configJson.Client(context.Background(), tok)
}

const LOCALHOST_PORT = 9023

// getTokenFromWeb gets a new OAuth2 token via the web-based authorization flow.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	config.RedirectURL = fmt.Sprintf("http://localhost:%d", LOCALHOST_PORT)
	// Generate the URL for the OAuth2 flow
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	// Start a local HTTP server to handle the redirect
	code := startLocalServer()

	// Exchange the authorization code for a token
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		log.Fatalf("Unable to create oauth token dir: %v", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func startLocalServer() string {
	codeChan := make(chan string)

	// Start a local HTTP server to capture the redirect URL
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Parse the authorization code from the query parameters
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		// Respond to let the user know the code has been captured
		w.Write([]byte("You can close this window now."))

		// Send the code to the channel
		codeChan <- code
	})

	// Start the local HTTP server on port 8080
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", LOCALHOST_PORT), nil); err != nil {
			log.Fatalf("Failed to start local server: %v", err)
		}
	}()

	// Wait for the user to authorize and the server to handle the code
	fmt.Println("Waiting for authorization code...")
	code := <-codeChan
	return code
}
