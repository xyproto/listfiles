package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/binary"
	"github.com/xyproto/mime"
	"github.com/xyproto/mode"
)

const maxBinaryDetectionFileSize = 1024 * 1024 * 1024

var mi *mime.Reader

// FileTypeInfo contains comprehensive information about a file's type
type FileTypeInfo struct {
	Mode        mode.Mode
	IsBinary    bool
	Description string
	TypeColor   string
	NameColor   string
	LineCount   int // -1 if not counted
}

// DetectFileType performs comprehensive file type detection similar to Orbiton
// If data is nil, detection will be based only on the filename
func DetectFileType(filename string, fileInfo os.FileInfo, data []byte) FileTypeInfo {
	var (
		m           mode.Mode
		isBinary    bool
		description string
		typeColor   string
		nameColor   string
		lineCount   = -1
	)

	// Check if it's a directory first
	if fileInfo.IsDir() {
		return FileTypeInfo{
			Mode:        mode.Blank,
			Description: "Directory",
			TypeColor:   "magenta",
			NameColor:   "lightcyan",
			LineCount:   -1,
		}
	}

	// Initial mode detection based on filename/extension
	m = mode.Detect(filename)

	// If we have file contents, do deeper analysis
	if data != nil {
		// Check if file is binary
		isBinary = binary.Data(data)

		if !isBinary {
			// Count lines for text files
			lineCount = bytes.Count(data, []byte{'\n'})

			// If mode is blank or certain special cases, try content detection
			if m == mode.Blank || m == mode.Prolog || m == mode.Config ||
				(m == mode.Markdown && !strings.HasSuffix(strings.ToLower(filename), ".md")) {

				// Look at first line/chunk for content detection
				var firstChunk []byte
				if idx := bytes.IndexByte(data, '\n'); idx > 0 {
					firstChunk = data[:idx]
				} else {
					firstChunk = data
				}

				// Limit chunk size for efficiency
				if len(firstChunk) > 512 {
					firstChunk = firstChunk[:512]
				}

				// Try to detect from content
				if newMode, found := mode.DetectFromContentBytes(m, firstChunk, func() []byte { return data }); found {
					m = newMode
				}
			}

			// Special case for Assembly files
			if m == mode.Assembly {
				if newMode, found := mode.DetectFromContentBytes(m, data, func() []byte { return data }); found {
					m = newMode
				}
			}

		}
	}

	if m == mode.Blank && fileInfo.Size() < maxBinaryDetectionFileSize {
		if data, err := os.ReadFile(filename); err == nil { // success
			isBinary = binary.Data(data)
		}
	}

	// Determine colors and description based on the detected type
	description, typeColor, nameColor = getTypeDescriptionAndColors(m, isBinary, fileInfo.IsDir())

	// Keep the colors but change the description if the file is empty
	if fileInfo.Size() == 0 {
		description = "Empty"
	}

	if description == "Unknown" {
		if mi == nil {
			mi = mime.New("testconf/mime.types", true)
		}
		mimeDescription := strings.TrimSpace(mi.Get(filepath.Ext(filename)))
		if mimeDescription != "" {
			description = mimeDescription
			if strings.Contains(description, "/") {
				fields := strings.SplitN(description, "/", 2)
				description = strings.TrimPrefix(fields[1], "x-")
				firstLetter := strings.ToUpper(string(description[0]))
				description = firstLetter + description[1:]
			}
		}
	}

	return FileTypeInfo{
		Mode:        m,
		IsBinary:    isBinary,
		Description: description,
		TypeColor:   typeColor,
		NameColor:   nameColor,
		LineCount:   lineCount,
	}
}

// getTypeDescriptionAndColors returns appropriate colors and description for the file type
func getTypeDescriptionAndColors(m mode.Mode, isBinary bool, isDir bool) (description, typeColor, nameColor string) {
	if isDir {
		return "Directory", "magenta", "lightcyan"
	}
	if isBinary {
		return "Binary", "red", "lightred"
	}
	switch m {
	case mode.Markdown, mode.Text, mode.ReStructured, mode.SCDoc, mode.ASCIIDoc:
		return m.String(), "cyan", "magenta"
	case mode.Config:
		return m.String(), "cyan", "yellow"
	case mode.Python, mode.Go, mode.Rust, mode.C, mode.Cpp:
		return m.String(), "cyan", "lightgreen"
	case mode.HTML, mode.CSS, mode.JavaScript:
		return m.String(), "cyan", "yellow"
	case mode.Shell, mode.Perl, mode.Ruby:
		return m.String(), "cyan", "lightred"
	case mode.Git:
		return m.String(), "cyan", "green"
	default:
		if m != mode.Blank {
			return m.String(), "cyan", "lightgreen"
		}
		return "Unknown", "gray", "white"
	}
}
