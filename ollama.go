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
		prompt := "Which command can be used on " + distrodetector.New().Name() + " to build this project? Only answer with the command, or NOPE.\n\n" + fileOverview
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
			sb.WriteString(fmt.Sprintf("Build %s, suggested by %s:\n", english.PluralWord(l, "command", ""), ollamaModel))
			if l > 1 {
				sb.WriteString("\n")
			}
			sb.WriteString(strings.Join(buildCommands, "\n") + "\n")

			if l > 0 && buildCommands[l-1] == "NOPE" {
				return "", fmt.Errorf("Ollama (%s) could not propose a suitable build command", ollamaModel)
			}

			*needsSeparator = true
		}
	}
	return sb.String(), nil
}
