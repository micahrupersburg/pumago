package config

import (
	"log"
	"os"
	"path/filepath"
)

func Dir() string {
	return filepath.Join(os.Getenv("HOME"), ".config/puma")
}
func BinDir() string {
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v\n", err)
	}

	return filepath.Dir(executable)
}
