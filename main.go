package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/xyproto/mode"
	"github.com/xyproto/textoutput"
)

func main() {
	const (
		path                   = "."
		respectIgnoreFiles     = true
		respectHiddenFiles     = true
		readFileSizeThreshold  = 1000 * 1024
		lineCountSizeThreshold = 100 * 1024
	)

	maxDepth := 1

	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil { // success
			maxDepth = n
		}
	}

	o := textoutput.New()

	findings, err := Examine(path, respectIgnoreFiles, respectHiddenFiles, maxDepth)
	if err != nil {
		o.ErrExit("FAIL:" + err.Error())
	}

	var (
		dirList  []string
		printed  bool
		printMap = make(map[time.Time]string)
	)

	// List regular files
	for _, fn := range findings.regularFiles {
		// If we don't have file info, just show the filename
		fInfo, ok := findings.infoMap[fn]
		if !ok {
			o.Println(fmt.Sprintf("<white>%s</white>", fn))
			printed = true
			continue
		}

		// Read file contents if it's small enough
		var fileContents []byte
		if fInfo.Size() < readFileSizeThreshold {
			if data, err := os.ReadFile(fn); err == nil {
				fileContents = data
			}
		}

		// Detect file type using contents if available
		typeInfo := DetectFileType(fn, fInfo, fileContents)

		// Generate size description
		var sizeDescription string
		if typeInfo.IsBinary || typeInfo.LineCount < 0 {
			sizeDescription = fmt.Sprintf("%d bytes", fInfo.Size())
		} else {
			sizeDescription = fmt.Sprintf("%d lines", typeInfo.LineCount)
		}

		// Format and print the output
		if typeInfo.Mode == mode.Blank && fInfo.IsDir() {
			dirList = append(dirList, fn)
		} else {
			modified := fInfo.ModTime()
			cell1 := fmt.Sprintf("<%s>%s</%s>", typeInfo.NameColor, fn, typeInfo.NameColor)
			cell2 := fmt.Sprintf("[<%s>%s</%s>]", typeInfo.TypeColor, typeInfo.Description, typeInfo.TypeColor)
			cell3 := TimeString(ok, modified, "lightyellow", "yellow", "white", "lightblue")
			cell4 := sizeDescription
			printMap[modified] = cell1 + ";" + cell2 + ";" + cell3 + ";" + cell4
		}
	}

	if l := len(printMap); l > 0 {
		keys := make([]time.Time, l, l)
		counter := 0
		for k := range printMap {
			keys[counter] = k
			counter++
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Before(keys[j])
		})
		// Print all the files, sorted by modification time
		for _, k := range keys {
			o.Println(printMap[k])
		}
		// Separation line
		o.Println()
	}

	// List directories
	sort.Strings(dirList)
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
			return errors.New("break") // break
		})

	}

	// Separation line
	//if printed {
	//o.Println()
	//printed = false
	//}
}
