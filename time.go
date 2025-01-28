package main

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

// TimeString returns a nicely formatted and colored string, with color tags.
// Set timeColor to an empty string to disable colors.
func TimeString(foundTime bool, modified time.Time, agoColor, timeColor, dateColor string) string {
	if !foundTime {
		return "-"
	}
	if elapsed := time.Since(modified); elapsed < time.Hour*24 {
		if timeColor == "" {
			return fmt.Sprintf("%s @%s", humanize.Time(modified), modified.Format("15:04:05"))
		}
		return fmt.Sprintf("<"+agoColor+">%s</"+agoColor+"> @<"+timeColor+">%s</"+timeColor+">", humanize.Time(modified), modified.Format("15:04:05"))
	}
	if timeColor == "" {
		return fmt.Sprintf("%s @%s", modified.Format("2006-01-02"), modified.Format("15:04:05"))
	}
	return fmt.Sprintf("<"+dateColor+">%s</"+dateColor+"> @<"+timeColor+">%s</"+timeColor+">", modified.Format("2006-01-02"), modified.Format("15:04:05"))
}
