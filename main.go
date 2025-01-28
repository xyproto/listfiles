package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/xyproto/mode"
	"github.com/xyproto/textoutput"
)

const (
	versionString        = "pal 0.2.2"
	defaultPath          = "."
	defaultMaxDepth      = 1
	defaultReadThreshold = 10 * 1042 * 1024 // 10 MiB file size
	defaultLineThreshold = 1 * 1024 * 1024  // 1 MiB file size
)

type Config struct {
	maxDepth              int
	path                  string
	showAll               bool
	respectIgnored        bool
	respectHidden         bool
	readFileSizeThreshold int64
	lineCountThreshold    int64
	ollama                bool
}

func parseHumanSize(sizeStr string) (int64, error) {
	// Try parsing as a plain number first
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size, nil
	}

	// Try parsing as a human-readable size
	size, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s (use bytes or units like K, M, G)", sizeStr)
	}
	return int64(size), nil
}

func processArgs(cfg *Config, args []string) error {
	// Handle optional path argument
	if len(args) > 0 {
		// Clean and make the path absolute if possible
		path, err := filepath.Abs(args[0])
		if err == nil {
			cfg.path = path
		} else {
			cfg.path = args[0] // Fallback to the provided path
		}
		args = args[1:] // Remove processed argument
	}

	// Handle optional depth argument
	if len(args) > 0 {
		if depth, err := strconv.Atoi(args[0]); err == nil {
			if depth < 0 {
				return fmt.Errorf("depth must be a non-negative number")
			}
			cfg.maxDepth = depth
			args = args[1:] // Remove processed argument
		}
	}

	// Check for unexpected arguments
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
	}

	// Update config based on flags
	if cfg.showAll {
		cfg.respectIgnored = false
		cfg.respectHidden = false
	}

	// Validate path exists
	if _, err := os.Stat(cfg.path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", cfg.path)
		}
		return fmt.Errorf("error accessing path: %s", err)
	}

	return nil
}

func NewRootCommand() *cobra.Command {
	cfg := &Config{
		maxDepth:              defaultMaxDepth,
		path:                  defaultPath,
		respectIgnored:        true,
		respectHidden:         true,
		readFileSizeThreshold: defaultReadThreshold,
		lineCountThreshold:    defaultLineThreshold,
		ollama:                false,
	}

	cmd := &cobra.Command{
		Use:   "pal [depth] [path]",
		Short: "List and analyze files in a directory",
		Long: `pal can list information about a directory

Example use:
  pal                     # Examine the current directory with depth 1
  pal . 2                 # Examine the current directory with depth 2
  pal /path/to/dir 2      # Examine the specified directory with depth 2
  pal /path/to/dir        # Examine the specified directory with depth 1`,
		Version: versionString,
		RunE: func(_ *cobra.Command, args []string) error {
			if err := processArgs(cfg, args); err != nil {
				return err
			}
			return run(cfg)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&cfg.showAll, "all", "a", false, "show all files (including hidden and ignored)")
	flags.BoolVarP(&cfg.ollama, "ollama", "o", false, "use ollama to suggest a build command")

	// Configure version flag
	cmd.SetVersionTemplate(versionString + "\n")

	return cmd
}

func run(cfg *Config) error {

	var (
		dirList        []string
		printMap       = make(map[time.Time]string)
		needsSeparator bool
		o              = textoutput.New()
		pi             strings.Builder
	)

	findings, err := Examine(cfg.path, cfg.respectIgnored, cfg.respectHidden, cfg.maxDepth)
	if err != nil {
		return fmt.Errorf("examination failed: %w", err)
	}

	// In the file processing loop, use cfg.readFileSizeThreshold:
	for _, fn := range findings.regularFiles {
		fInfo, ok := findings.infoMap[fn]
		if !ok {
			o.Println(fmt.Sprintf("<white>%s</white>", fn))
			needsSeparator = true
			continue
		}

		// Read file contents if it's small enough
		var fileContents []byte
		if fInfo.Size() < cfg.readFileSizeThreshold {
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
			if fn != "." {
				dirList = append(dirList, fn)
			}
		} else {
			modified := fInfo.ModTime()
			cell1 := fmt.Sprintf("<%s>%s</%s>", typeInfo.NameColor, fn, typeInfo.NameColor)
			cell2 := fmt.Sprintf("[<%s>%s</%s>]", typeInfo.TypeColor, typeInfo.Description, typeInfo.TypeColor)
			cell3 := TimeString(ok, modified, "lightyellow", "lightblue", "white")
			cell4 := sizeDescription
			printMap[modified] = cell1 + ";" + cell2 + ";" + cell3 + ";" + cell4

			// Project info, to be sent to Ollama
			pi.WriteString(fn + "\n")
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
		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(printMap[k])
			sb.WriteString("\n")
		}
		o.Print(sb.String())

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

		o.Printf("<yellow>Git URL:</yellow> <lightblue>%s</lightblue>\n", findings.git.URL)

		r, err := git.PlainOpen(cfg.path)
		if err != nil {
			return err
		}
		ref, err := r.Head()
		if err != nil {
			return err
		}

		now := time.Now() // time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
		oneYearAgo := time.Now().AddDate(-1, 0, 0)

		cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &oneYearAgo, Until: &now})
		if err != nil {
			return err
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
			return errors.New("break") // return nil instead to continue priting git commit messages
		})
	}

	// Ask Ollama what a sensible build command could be
	if cfg.ollama {
		if result, err := askOllama(&needsSeparator, pi.String()); err == nil { // success
			o.Println(result)
		}
	}

	return nil
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
