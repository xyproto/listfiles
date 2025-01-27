package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

type Git struct {
	URL string
}

// transformGitURL tries to transform a git@github.com:xyproto/pal style URL
// to a https://github.com/xyproto/pal style URL.
func transformGitURL(url string) string {
	if !strings.HasPrefix(url, "git@") {
		return url
	}
	rest := url[4:]
	if !strings.Contains(rest, ":") {
		return url
	}
	return "https://" + strings.Replace(rest, ":", "/", 1)
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
					git.URL = transformGitURL(string(value))
					break // TODO: Don't skip the rest of the "url = ..." lines, but try to find the main one (the one for the main or master branch)
				}
			}
		}
	}

	return &git, nil
}
