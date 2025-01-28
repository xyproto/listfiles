package main

import (
	"fmt"
	"time"
)

func TimeString(foundTime bool, modified time.Time, agoColor, recentTimeColor, oldTimeColor, dateColor string) string {
	if !foundTime {
		return "-"
	}
	if elapsed := time.Since(modified); elapsed < time.Hour*24 {
		return fmt.Sprintf("<"+agoColor+">%s ago @</"+agoColor+"><"+recentTimeColor+">%s</"+recentTimeColor+">", formatElapsed(elapsed), modified.Format("15:04:05"))
	}
	return fmt.Sprintf("<"+oldTimeColor+">%s</"+oldTimeColor+">, <"+dateColor+">%s</"+dateColor+">", modified.Format("2006-01-02"), modified.Format("15:04:05"))
}
