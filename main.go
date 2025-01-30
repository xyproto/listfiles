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
	versionString        = "pal 0.2.5"
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

func (cfg *Config) AnalyzeFiles(ob *strings.Builder, findings *Findings, needsSeparator *bool) error {
	findings.printMap = make(map[time.Time]string)
	if *needsSeparator {
		ob.WriteString("\n")
		*needsSeparator = false
	}
	// In the file processing loop, use cfg.readFileSizeThreshold:
	for _, fn := range findings.regularFiles {
		fInfo, ok := findings.infoMap[fn]
		if !ok {
			ob.WriteString(fmt.Sprintf("<white>%s</white>\n", fn))
			*needsSeparator = true
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
				findings.dirList = append(findings.dirList, fn)
			}
		} else {
			modified := fInfo.ModTime()
			cell1 := fmt.Sprintf("<%s>%s</%s>", typeInfo.NameColor, fn, typeInfo.NameColor)
			cell2 := fmt.Sprintf("[<%s>%s</%s>]", typeInfo.TypeColor, typeInfo.Description, typeInfo.TypeColor)
			cell3 := TimeString(ok, modified, "lightyellow", "lightblue", "white")
			cell4 := sizeDescription
			findings.printMap[modified] = cell1 + ";" + cell2 + ";" + cell3 + ";" + cell4
			findings.fileList = append(findings.fileList, fn)
		}
	}
	return nil
}

func (cfg *Config) ListFiles(ob *strings.Builder, findings *Findings, needsSeparator *bool) error {
	// List files, if any
	if l := len(findings.printMap); l > 0 {
		if *needsSeparator {
			ob.WriteString("\n")
			*needsSeparator = false
		}
		keys := make([]time.Time, l, l)
		counter := 0
		for k := range findings.printMap {
			keys[counter] = k
			counter++
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Before(keys[j])
		})
		// Print out the full line of info for a single file, from the printMap
		for _, k := range keys {
			ob.WriteString(findings.printMap[k])
			ob.WriteString("\n")
		}
		*needsSeparator = true
	}
	return nil
}

func (cfg *Config) ListDirs(ob *strings.Builder, findings *Findings, needsSeparator *bool) {
	// List directories, if any
	if len(findings.dirList) > 0 {
		if *needsSeparator {
			ob.WriteString("\n")
			*needsSeparator = false
		}
		sort.Strings(findings.dirList)
		for _, dirName := range findings.dirList {
			ob.WriteString(fmt.Sprintf("[<magenta>dir</magenta>] <lightcyan>%s</lightcyan><lightgreen>/</lightgreen>\n", dirName))
		}
		*needsSeparator = true
	}
}

func (cfg *Config) IgnoredFiles(ob *strings.Builder, findings *Findings, needsSeparator *bool) {
	// Ignored files
	if ignoredLen := len(findings.ignoredFiles); ignoredLen > 0 {
		if *needsSeparator {
			ob.WriteString("\n")
			*needsSeparator = false
		}

		ob.WriteString(fmt.Sprintf("</white>There %s %d ignored %s.</white>\n", english.PluralWord(ignoredLen, "is", "are"), ignoredLen, english.PluralWord(ignoredLen, "file", "")))

		*needsSeparator = true
	}
}

func (cfg *Config) LatestGitCommitThisYear(ob *strings.Builder, findings *Findings, needsSeparator *bool) error {
	// Git URL
	if findings.git != nil {
		if *needsSeparator {
			ob.WriteString("\n")
			*needsSeparator = false
		}

		ob.WriteString(fmt.Sprintf("<yellow>Git URL:</yellow> <lightblue>%s</lightblue>\n", findings.git.URL))

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
				if *needsSeparator {
					ob.WriteString("\n")
					*needsSeparator = false
				}

				commitTextLines := strings.Split(logEntryAsString, "\n")
				ob.WriteString(GitHighlightLines(commitTextLines))

				*needsSeparator = true
			}
			return errors.New("break") // return nil instead to continue priting git commit messages
		})
	}

	return nil
}

func (cfg *Config) OllamaBuildCommand(ob *strings.Builder, findings *Findings, needsSeparator *bool) error {
	// Ask Ollama what a sensible build command could be
	if cfg.ollama {
		if *needsSeparator {
			ob.WriteString("\n")
			*needsSeparator = false
		}

		model, err := NewModel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nCould not connect to Ollama: %v\n", err)
			return nil // don't report this as an error on top of this
		}
		if result, err := model.GetBuildCommand(strings.Join(findings.fileList, "\n")); err == nil { // success
			ob.WriteString(result + "\n")
		}

		*needsSeparator = false
	}
	return nil
}

func run(cfg *Config) error {
	var (
		needsSeparator bool
		ob             strings.Builder // output string
	)

	findings, err := Examine(cfg.path, cfg.respectIgnored, cfg.respectHidden, cfg.maxDepth)
	if err != nil {
		return fmt.Errorf("file search failed: %v", err)
	}
	if err := cfg.AnalyzeFiles(&ob, findings, &needsSeparator); err != nil {
		return fmt.Errorf("analyzing files failed: %v", err)
	}

	cfg.IgnoredFiles(&ob, findings, &needsSeparator)

	cfg.ListDirs(&ob, findings, &needsSeparator)

	cfg.ListFiles(&ob, findings, &needsSeparator)

	cfg.LatestGitCommitThisYear(&ob, findings, &needsSeparator)

	cfg.OllamaBuildCommand(&ob, findings, &needsSeparator)

	textoutput.New().Print(ob.String())

	return nil
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
}
