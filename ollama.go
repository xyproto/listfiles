package main

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize/english"
	"github.com/xyproto/distrodetector"
	"github.com/xyproto/ollamaclient/v2"
	"github.com/xyproto/usermodel"
)

func askOllama(needsSeparator *bool, fileOverview string) (string, error) {
	var sb strings.Builder
	ollamaModel := usermodel.GetCodeModel()
	oc := ollamaclient.New(ollamaModel)
	oc.Verbose = false
	if err := oc.PullIfNeeded(); err == nil { // success
		if *needsSeparator {
			sb.WriteString("\n")
			*needsSeparator = false
		}
		distroName := distrodetector.New().Name()
		prompt := fmt.Sprintf("You are an expert %s developer. Which command can the user run to build or compile a project that has the following files:\n\n%s\n\nAnswer with a command that a script can run directly (no commentary), or just say true.", distroName, fileOverview)
		output, err := oc.GetOutput(prompt)
		if err != nil {
			return "", err
		}
		lines := strings.Split(ollamaclient.Massage(output, false), "\n")
		buildCommands := make([]string, 0)
		for _, line := range lines {
			if strings.HasPrefix(line, "```") || strings.HasSuffix(line, "```") {
				continue
			}
			buildCommands = append(buildCommands, strings.TrimSuffix(strings.TrimPrefix(line, "`"), "`"))
		}
		if l := len(buildCommands); l > 0 {
			sb.WriteString(fmt.Sprintf("<lightblue>Build %s, suggested by</lightblue> <lightyellow>%s</lightyellow><lightblue>:</lightblue>\n", english.PluralWord(l, "command", ""), ollamaModel))
			if l > 1 {
				sb.WriteString("\n")
			}
			sb.WriteString(strings.Join(buildCommands, "\n") + "\n")

			if l > 0 && strings.TrimSpace(buildCommands[l-1]) == "true" {
				return "", fmt.Errorf("Ollama (%s) could not propose a suitable build command", ollamaModel)
			}

			*needsSeparator = true
		}
	}
	return sb.String(), nil
}
