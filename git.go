package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xyproto/files"
)

type Git struct {
	URL string
}

func NewGit(path string) (*Git, error) {
	if !files.IsDir(path) {
		return nil, fmt.Errorf("not a .git directory: %s", path)
	}
	configData, err := os.ReadFile(filepath.Join(path, "config"))
	if err != nil {
		return nil, fmt.Errorf("could not read .git/config: %v", err)
	}
	var git Git
	byteLines := bytes.Split(configData, []byte("\n"))
	for _, byteLine := range byteLines {
		trimmedByteLine := bytes.TrimSpace(byteLine)
		if bytes.Contains(trimmedByteLine, []byte("=")) {
			fields := bytes.SplitN(trimmedByteLine, []byte("="), 2)
			if len(fields) == 2 {
				key := bytes.TrimSpace(fields[0])
				value := bytes.TrimSpace(fields[1])
				if string(key) == "url" {
					git.URL = string(value)
					break // TODO: Don't skip the rest of the "url = ..." lines, but try to find the main one (the one for the main or master branch)
				}
			}
		}
	}

	return &git, nil
}
