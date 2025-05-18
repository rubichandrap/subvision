package utils

import (
	"log"
	"os"
)

func EnsureDirs(paths ...string) {
	for _, path := range paths {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			log.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}
}
