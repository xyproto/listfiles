package main

import (
	"fmt"
	"log"
)

func main() {
	const path = "."
	findings, err := examine(path)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Findings:")
	fmt.Printf("%s has git? %v\n", path, findings.git != nil)
	if findings.git != nil {
		fmt.Printf("git URL: %s\n", findings.git.URL)
	}
	fmt.Println("Regular files:")
	for _, fn := range findings.regularFiles {
		fmt.Println(fn)
	}
}
