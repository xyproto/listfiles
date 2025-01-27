package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

func main() {
	const path = "."
	findings, err := examine(path)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Findings:")

	hasGit := findings.git != nil

	fmt.Printf("%s has git? %v\n", path, hasGit)

	if hasGit {
		fmt.Printf("git URL: %s\n", findings.git.URL)

		fmt.Println("git clone into memory")
		r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
			URL: findings.git.URL,
		})
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("git log")
		ref, err := r.Head()
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("get the commit history")

		since := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
		until := time.Date(2019, 7, 30, 0, 0, 0, 0, time.UTC)
		cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &since, Until: &until})
		if err != nil {
			log.Fatalln(err)
		}

		err = cIter.ForEach(func(c *object.Commit) error {
			fmt.Println(c)
			return nil
		})
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Println("Regular files:")
	for _, fn := range findings.regularFiles {
		fmt.Println(fn)
	}
}
