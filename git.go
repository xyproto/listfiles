package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
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

// GitHighlightLines applies syntax highlighting for a git log line
func GitHighlightLines(lines []string) string {
	var sb strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "commit ") {
			fields := strings.SplitN(line, " ", 2) // must work, already checked for " "
			sb.WriteString("<red>commit</red> <white>")
			sb.WriteString(strings.TrimSpace(fields[1])) // must work, len(fields) is now 2
			sb.WriteString("</white>\n")
		} else if strings.HasPrefix(line, "Author:") {
			fields := strings.SplitN(line, ":", 2) // must work, already checked for ":"
			sb.WriteString("<blue>")
			sb.WriteString(strings.TrimSpace(fields[1])) // must work, len(fields) is now 2
			sb.WriteString("</blue>")
		} else if strings.HasPrefix(line, "Date:") {
			fields := strings.SplitN(line, ":", 2)     // must work, already checked for ":"
			timeString := strings.TrimSpace(fields[1]) // must work, len(fields) is now 2
			// Example git time: Mon Jan 27 17:37:49 2025 +0100
			const gitTimeFormatString = "Mon Jan 2 15:04:05 2006 -0700"            // magical numbers, see the time package documentation
			if t, err := time.Parse(gitTimeFormatString, timeString); err == nil { // success
				sb.WriteString(", <yellow>")
				if elapsed := time.Since(t); elapsed < time.Hour*24 {
					sb.WriteString(humanize.Time(t))
				} else {
					sb.WriteString(t.Format("2006-01-02 15:04:05"))
				}
				sb.WriteString("</yellow>")
			} else {
				sb.WriteString("<yellow>")
				sb.WriteString(line)
				sb.WriteString("</yellow>")
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("<green>")
			sb.WriteString(line)
			sb.WriteString("</green>")
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
