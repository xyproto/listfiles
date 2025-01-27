package main

import (
	"fmt"
	"time"
)

func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		h := d / time.Hour
		m := (d % time.Hour) / time.Minute
		return fmt.Sprintf("%dh%dm", h, m)
	}
	days := d / (24 * time.Hour)
	h := (d % (24 * time.Hour)) / time.Hour
	return fmt.Sprintf("%dd%dh", days, h)
}
