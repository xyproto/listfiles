package main

import (
	"errors"
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

func main() {

	const (
		path               = "."
		respectIgnoreFiles = true // Ignore filenames mentioned in .ignore or .gitignore
		respectHiddenFiles = true // Ignore filenames starting with "."
		maxDepth           = 1
	)

	o := textoutput.New()

	findings, err := Examine(path, respectIgnoreFiles, respectHiddenFiles, maxDepth)
	if err != nil {
		o.ErrExit("FAIL:" + err.Error())
	}

	var (
		atLeastOneRegularFile bool
		dirList               []string
	)

	// Regular files
	for _, fn := range findings.regularFiles {
		var (
			isBinary  bool
			isDir     bool
			m         mode.Mode
			modified  time.Time
			foundTime bool
		)
		if fInfo, ok := findings.infoMap[fn]; ok {
			foundTime = true
			modified = fInfo.ModTime()
			isDir = fInfo.IsDir()
			if !isDir {
				atLeastOneRegularFile = true
			}
		}
		m = mode.Detect(fn)
		if data, err := os.ReadFile(fn); err == nil { // success
			isBinary = binary.Data(data)
		}
		if isBinary {
			o.Printf("<lightred>%s</lightred> [<red>binary</red>]", fn)
		} else if isDir {
			dirList = append(dirList, fn)
		} else if m == mode.Blank {
			o.Printf("<white>%s</white>", fn)
		} else if m == mode.Markdown || m == mode.Text || m == mode.ReStructured || m == mode.SCDoc || m == mode.ASCIIDoc {
			o.Printf("<magenta>%s</magenta>", fn)
		} else {
			o.Printf("<lightgreen>%s</lightgreen>", fn)
		}
		if !isDir {
			if m != mode.Blank && !isBinary {
				o.Printf(" [<cyan>%s</cyan>]", m)
			}
			if foundTime {
				elapsed := time.Since(modified)
				if elapsed < time.Hour*24 {
					o.Printf(" <lightyellow>%s ago @</lightyellow><yellow>%s</lightblue>", formatElapsed(elapsed), modified.Format("15:04:05"))
				} else {
					o.Printf(" <white>%s</white>, <lightblue>%s</lightblue>", modified.Format("2006-01-02"), modified.Format("15:04:05"))
				}
			}
			o.Println()
		}
	}
	if atLeastOneRegularFile {
		o.Println()
	}

	// Output directories

	for _, dirName := range dirList {
		o.Printf("[<magenta>dir</magenta>] <lightcyan>%s</lightcyan><lightgreen>/</lightgreen>\n", dirName)
	}
	if len(dirList) > 0 {
		o.Println()
	}

	// Ignored files
	ignoredLen := len(findings.ignoredFiles)
	if ignoredLen == 1 {
		o.Println("<white>There is also one ignored file.</white>\n")
	} else if ignoredLen > 1 {
		o.Printf("</white>There are also %d ignored files.</white>\n\n", ignoredLen)
	}

	// Git URL
	if findings.git != nil {
		o.Printf("<white>Git URL:</white> <red>%s</red>\n", findings.git.URL)
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

		o.Println()

		// ignore err here because we want to break the loop early
		_ = cIter.ForEach(func(c *object.Commit) error {
			logEntryAsString := strings.TrimRightFunc(c.String(), unicode.IsSpace)
			o.Printf("<yellow>%s</yellow>\n", logEntryAsString)
			//return nil // continue
			return errors.New("stop") // break
		})

		//if err != nil {
		//o.ErrExit("FILA " + err.Error())
		//log.Fatalln(err)
		//}
	}
}
