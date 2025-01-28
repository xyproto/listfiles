package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/xyproto/binary"
	"github.com/xyproto/mode"
	"github.com/xyproto/textoutput"
)

func TypeDescriptionAndColors(m mode.Mode, isDir, isBinary bool) (string, string, string) {
	if !isDir && m != mode.Blank && !isBinary {
		return m.String(), "cyan", "lightgreen"
	} else if isBinary {
		return "Binary", "red", "lightred"
	}
	switch m {
	case mode.Blank:
		return "Unknown", "gray", "white"
	case mode.Markdown, mode.Text, mode.ReStructured, mode.SCDoc, mode.ASCIIDoc:
		return m.String(), "cyan", "magenta"
	case mode.Config:
		return m.String(), "cyan", "yellow"
	}
	return "Unknown", "gray", "white"
}

func TimeString(foundTime bool, modified time.Time, agoColor, recentTimeColor, oldTimeColor, dateColor string) string {
	if !foundTime {
		return "-"
	}
	if elapsed := time.Since(modified); elapsed < time.Hour*24 {
		return fmt.Sprintf("<"+agoColor+">%s ago @</"+agoColor+"><"+recentTimeColor+">%s</"+recentTimeColor+">", formatElapsed(elapsed), modified.Format("15:04:05"))
	}
	return fmt.Sprintf("<"+oldTimeColor+">%s</"+oldTimeColor+">, <"+dateColor+">%s</"+dateColor+">", modified.Format("2006-01-02"), modified.Format("15:04:05"))
}

func main() {

	const (
		path               = "."
		respectIgnoreFiles = true // Ignore filenames mentioned in .ignore or .gitignore
		respectHiddenFiles = true // Ignore filenames starting with "."
		maxDepth           = 1

		readFileSizeThreshold  = 1000 * 1024 // Don't read in files larger than 100k
		lineCountSizeThreshold = 100 * 1024  // Don't start counting lines of text files larger than 10k
	)

	o := textoutput.New()

	findings, err := Examine(path, respectIgnoreFiles, respectHiddenFiles, maxDepth)
	if err != nil {
		o.ErrExit("FAIL:" + err.Error())
	}

	var (
		dirList []string
		printed bool
	)

	// List regular files
	for _, fn := range findings.regularFiles {
		var (
			isBinary         bool
			isDir            bool
			m                mode.Mode
			modified         time.Time
			foundTimeAndSize bool
			size             int64
		)
		if fInfo, ok := findings.infoMap[fn]; ok {
			foundTimeAndSize = true
			modified = fInfo.ModTime()
			isDir = fInfo.IsDir()
			size = fInfo.Size()
		}
		m = mode.Detect(fn)

		sizeDescription := "too large to analyze"
		if size < readFileSizeThreshold {
			if data, err := os.ReadFile(fn); err == nil { // success
				if m == mode.Blank {
					m = mode.SimpleDetectBytes(data)
				}
				isBinary = binary.Data(data)
				if isBinary {
					sizeDescription = fmt.Sprintf("%d bytes", size)
				} else if size < lineCountSizeThreshold {
					// TODO: Detect if there is a trailing newline for an improved line count
					lineCount := bytes.Count(data, []byte{'\n'})
					sizeDescription = fmt.Sprintf("%d lines", lineCount)
				}
			}
		}

		description, typeColor, fileColor := TypeDescriptionAndColors(m, isDir, isBinary)

		// Print the gathered data for this file as a colored line of text, not for directories

		if isDir {
			dirList = append(dirList, fn)
		} else {
			cell1 := fmt.Sprintf("[<"+typeColor+">%s</"+typeColor+">]", description)
			cell2 := fmt.Sprintf("<"+fileColor+">%s</"+fileColor+">", fn)
			cell3 := TimeString(foundTimeAndSize, modified, "lightyellow", "yellow", "white", "lightblue")
			cell4 := sizeDescription
			o.Println(cell2 + ";" + cell1 + ";" + cell3 + ";" + cell4)
			printed = true
		}
	}

	// Separation line
	if printed {
		o.Println()
		printed = false
	}

	// List directories
	for _, dirName := range dirList {
		o.Printf("[<magenta>dir</magenta>] <lightcyan>%s</lightcyan><lightgreen>/</lightgreen>\n", dirName)
		printed = true
	}

	// Separation line
	if printed {
		o.Println()
		printed = false
	}

	// Ignored files
	ignoredLen := len(findings.ignoredFiles)
	if ignoredLen == 1 {
		o.Println("<white>There is also one ignored file.</white>\n")
		printed = true
	} else if ignoredLen > 1 {
		o.Printf("</white>There are also %d ignored files.</white>\n", ignoredLen)
		printed = true
	}

	// Separation line
	if printed {
		o.Println()
		printed = false
	}

	// Git URL
	if findings.git != nil {
		o.Printf("<white>Git URL:</white> <red>%s</red>\n", findings.git.URL)
		printed = true
	}

	// Separation line
	if printed {
		o.Println()
		printed = false
	}

	// Last entry in the git log
	if findings.git != nil {
		//r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{URL: findings.git.URL})
		r, err := git.PlainOpen(path)
		if err != nil {
			o.ErrExit("FAIL:" + err.Error())
		}
		ref, err := r.Head()
		if err != nil {
			o.ErrExit("FAIL:" + err.Error())
		}

		now := time.Now() // time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
		oneYearAgo := time.Now().AddDate(-1, 0, 0)

		cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &oneYearAgo, Until: &now})
		if err != nil {
			o.ErrExit("FAIL " + err.Error())
		}

		// ignore err here because we want to break the loop early
		_ = cIter.ForEach(func(c *object.Commit) error {
			logEntryAsString := strings.TrimRightFunc(c.String(), unicode.IsSpace)
			o.Print(GitHighlightLines(strings.Split(logEntryAsString, "\n")))
			printed = true
			//return nil // continue
			return errors.New("stop") // break
		})

		//if err != nil {
		//o.ErrExit("FILA " + err.Error())
		//log.Fatalln(err)
		//}
	}

	// Separation line
	if printed {
		o.Println()
		printed = false
	}
}
