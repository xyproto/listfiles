package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/xyproto/binary"
	"github.com/xyproto/textoutput"
)

func main() {
	o := textoutput.New()

	const path = "."

	// Examine the given path
	fmt.Printf("Examining %s...", path)
	const respectIgnoreFiles = false
	findings, err := examine(path, respectIgnoreFiles)
	if err != nil {
		fmt.Println(" FAIL")
		log.Fatalln(err)
	}
	fmt.Println(" OK")

	// Regular files
	fmt.Println("Regular files:")
	for _, fn := range findings.regularFiles {
		isBinary := false
		if data, err := os.ReadFile(fn); err == nil { // success
			isBinary = binary.Data(data)
		}
		if isBinary {
			o.Printf("<lightred>%s</lightred>\n", fn)
		} else {
			o.Printf("<lightgreen>%s</lightgreen>\n", fn)
		}
	}
	o.Println()

	// Ignored files
	ignoredLen := len(findings.ignoredFiles)
	if ignoredLen == 1 {
		fmt.Printf("There is also %d ignored file.\n", ignoredLen)
	} else {
		fmt.Printf("There are also %d ignored files.\n", ignoredLen)
	}
	o.Println()

	// Git URL
	if findings.git != nil {
		o.Printf("<white>Git URL:</white> <red>%s</red>\n", findings.git.URL)
	}
	o.Println()

	// Last entry in the git log
	if findings.git != nil {
		fmt.Printf("Retrieving the git log...")
		//r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{URL: findings.git.URL})
		r, err := git.PlainOpen(path)
		if err != nil {
			o.ErrExit("FAIL:" + err.Error())
		}
		ref, err := r.Head()
		if err != nil {
			o.ErrExit("FAIL:" + err.Error())
		}

		now := time.Now() // time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
		oneYearAgo := time.Now().AddDate(-1, 0, 0)

		cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &oneYearAgo, Until: &now})
		if err != nil {
			o.ErrExit("FAIL " + err.Error())
		}

		fmt.Println()

		// ignore err here because we want to break the loop early
		_ = cIter.ForEach(func(c *object.Commit) error {
			o.Printf("<yellow>%s</yellow>\n", c)
			//return nil // continue
			return errors.New("stop") // break
		})

		//if err != nil {
		//o.ErrExit("FILA " + err.Error())
		//log.Fatalln(err)
		//}
		fmt.Println()
	}

	o.Println("<lightblue>Done.</lightblue>")
}
