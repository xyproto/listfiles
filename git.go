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
			fields := strings.SplitN(line, " ", 2) // already checked for " "
			sb.WriteString("<red>commit</red> <white>")
			sb.WriteString(strings.TrimSpace(fields[1])) // len(fields) is 2
			sb.WriteString("</white>\n")
		} else if strings.HasPrefix(line, "Author:") {
			fields := strings.SplitN(line, ":", 2)            // already checked for ":"
			nameAndMaybeEmail := strings.TrimSpace(fields[1]) // len(fields) is 2
			if strings.Contains(nameAndMaybeEmail, "@") && strings.Contains(nameAndMaybeEmail, "<") {
				fields = strings.SplitN(nameAndMaybeEmail, "<", 2)
				name := strings.TrimSpace(fields[0])
				email := strings.TrimSpace(fields[1])
				if strings.Contains(email, ">") {
					fields = strings.SplitN(email, ">", 2)
					email = strings.TrimSpace(fields[0])
				}
				sb.WriteString("<lightgreen>")
				sb.WriteString(name)
				sb.WriteString("</lightgreen> ")
				sb.WriteString("<red><</red>") // red <
				if strings.Contains(email, "@") {
					fields = strings.SplitN(email, "@", 2)
					username := strings.TrimSpace(fields[0])
					host := strings.TrimSpace(fields[1])
					sb.WriteString("<lightblue>")
					sb.WriteString(username)
					sb.WriteString("</lightblue><red>@</red><lightblue>")
					sb.WriteString(host)
					sb.WriteString("</lightblue>")
				} else {
					sb.WriteString("<lightblue>")
					sb.WriteString(email)
					sb.WriteString("</lightblue>")
				}
				sb.WriteString("<red>></red>") // red <
			} else {
				sb.WriteString("<blue>")
				sb.WriteString(nameAndMaybeEmail)
				sb.WriteString("</blue>")
			}
		} else if strings.HasPrefix(line, "Date:") {
			sb.WriteString(" ")
			fields := strings.SplitN(line, ":", 2)     // already checked for ":"
			timeString := strings.TrimSpace(fields[1]) // len(fields) is 2
			// Example git time: Mon Jan 27 17:37:49 2025 +0100
			const gitTimeFormatString = "Mon Jan 2 15:04:05 2006 -0700"            // magical numbers, see the time package documentation
			if t, err := time.Parse(gitTimeFormatString, timeString); err == nil { // success
				if elapsed := time.Since(t); elapsed < time.Hour*24*7 {
					sb.WriteString("<yellow>")
					sb.WriteString(humanize.Time(t))
					sb.WriteString("</yellow>")
					if elapsed < time.Hour*24 {
						sb.WriteString(" @<lightblue>")
						sb.WriteString(t.Format("15:04:05"))
						sb.WriteString("</lightblue>")
					}
				} else {
					sb.WriteString("<yellow>")
					sb.WriteString(t.Format("2006-01-02 15:04:05"))
					sb.WriteString("</yellow>")
				}
			} else {
				sb.WriteString("<yellow>")
				sb.WriteString(line)
				sb.WriteString("</yellow>")
			}
			sb.WriteString("\n")
		} else {
			// Without color
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
