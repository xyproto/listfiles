package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/iafan/cwalk"
	"github.com/xyproto/files"
)

type Findings struct {
	git          *Git
	mut          sync.Mutex
	regularFiles []string
	ignoredFiles []string
	infoMap      map[string]os.FileInfo
}

func NewFindings() *Findings {
	var findings Findings
	findings.regularFiles = make([]string, 0)
	findings.ignoredFiles = make([]string, 0)
	findings.infoMap = make(map[string]os.FileInfo)
	return &findings
}

func SplitPath(path string) []string {
	return strings.Split(path, string(filepath.Separator))
}

func examine(path string) (*Findings, error) {
	if !files.IsDir(path) {
		return nil, fmt.Errorf("not a path: %s", path)
	}

	findings := NewFindings()

	walkFunc := func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("os.Lstat error: %v", err)
		}
		if path == "" {
			return nil // skip
		}
		parts := SplitPath(path)
		if len(parts) == 0 {
			return fmt.Errorf("no path given: %s", path)
		}
		head := strings.ToLower(parts[0])
		if head == "vendor" {
			// Store the ignored file
			go func() {
				findings.mut.Lock()
				findings.ignoredFiles = append(findings.ignoredFiles, path)
				findings.infoMap[path] = fileInfo
				findings.mut.Unlock()
			}()
			return nil // skip
		}
		if head == ".git" {
			// Store the ignored file
			go func() {
				findings.mut.Lock()
				findings.ignoredFiles = append(findings.ignoredFiles, path)
				findings.infoMap[path] = fileInfo
				findings.mut.Unlock()
			}()
			// Find and store Git info
			go func() {
				git, err := NewGit(head) // pass in the path to the .git directory
				if err != nil {
					return // don't store the git struct in the findings
				}
				findings.mut.Lock()
				findings.git = git
				findings.mut.Unlock()
			}()
			return nil // skip
		}
		// Store a regular file
		go func() {
			findings.mut.Lock()
			findings.regularFiles = append(findings.regularFiles, path)
			findings.infoMap[path] = fileInfo
			findings.mut.Unlock()
		}()
		return nil
	}

	err := cwalk.Walk(path, walkFunc)
	if err != nil {
		return nil, err
	}
	return findings, nil
}
