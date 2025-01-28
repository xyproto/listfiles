package main

import "strings"

// GitHighlightLine applies syntax highlighting for a git log line
func GitHighlightLines(lines []string) string {
	var sb strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "commit") {
			sb.WriteString("<red>")
			sb.WriteString(line)
			sb.WriteString("</red>")
		} else if strings.HasPrefix(line, "Author:") {
			sb.WriteString("<blue>")
			sb.WriteString(line)
			sb.WriteString("</blue>")
		} else if strings.HasPrefix(line, "Date:") {
			sb.WriteString("<yellow>")
			sb.WriteString(line)
			sb.WriteString("</yellow>")
		} else {
			sb.WriteString("<green>")
			sb.WriteString(line)
			sb.WriteString("</green>")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
