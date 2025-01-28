package main

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

func TimeString(foundTime bool, modified time.Time, agoColor, timeColor, dateColor string) string {
	if !foundTime {
		return "-"
	}
	if elapsed := time.Since(modified); elapsed < time.Hour*24 {
		return fmt.Sprintf("<"+agoColor+">%s</"+agoColor+"> @<"+timeColor+">%s</"+timeColor+">", humanize.Time(modified), modified.Format("15:04:05"))
	}
	return fmt.Sprintf("<"+dateColor+">%s</"+dateColor+"> @<"+timeColor+">%s</"+timeColor+">", modified.Format("2006-01-02"), modified.Format("15:04:05"))
}
