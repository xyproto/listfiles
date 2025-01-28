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

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
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
		dirList        []string
		printMap       = make(map[time.Time]string)
		needsSeparator bool
	)

	// Collect info about regular files
	for _, fn := range findings.regularFiles {
		// If we don't have file info, just show the filename
		fInfo, ok := findings.infoMap[fn]
		if !ok {
			o.Println(fmt.Sprintf("<white>%s</white>", fn))
			needsSeparator = true
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
			sizeDescription = humanize.IBytes(uint64(fInfo.Size()))
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
			cell3 := TimeString(ok, modified, "lightyellow", "lightblue", "white")
			cell4 := sizeDescription
			printMap[modified] = cell1 + ";" + cell2 + ";" + cell3 + ";" + cell4
		}
	}

	// List files, if any
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
		needsSeparator = true
	}

	// List directories, if any
	if len(dirList) > 0 {
		if needsSeparator {
			o.Println()
			needsSeparator = false
		}
		sort.Strings(dirList)
		for _, dirName := range dirList {
			o.Printf("[<magenta>dir</magenta>] <lightcyan>%s</lightcyan><lightgreen>/</lightgreen>\n", dirName)
		}
		needsSeparator = true
	}

	// Ignored files
	if ignoredLen := len(findings.ignoredFiles); ignoredLen > 0 {
		if needsSeparator {
			o.Println()
			needsSeparator = false
		}

		o.Printf("</white>There %s also %d ignored %s.</white>\n", english.PluralWord(ignoredLen, "is", "are"), ignoredLen, english.PluralWord(ignoredLen, "file", ""))

		needsSeparator = true
	}

	// Git URL
	if findings.git != nil {
		if needsSeparator {
			o.Println()
			needsSeparator = false
		}

		o.Printf("<white>Git URL:</white> <red>%s</red>\n", findings.git.URL)

		needsSeparator = true
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
			if len(logEntryAsString) > 0 {
				if needsSeparator {
					o.Println()
					needsSeparator = false
				}

				commitTextLines := strings.Split(logEntryAsString, "\n")
				o.Print(GitHighlightLines(commitTextLines))

				needsSeparator = true
			}
			//return nil // continue
			return errors.New("break") // break
		})

	}
}
