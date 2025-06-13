package main

import (
	"flag"
	"fmt"
	"os"

	"replbac/internal/cmd"
)

func main() {
	var outputFile string
	flag.StringVar(&outputFile, "output", "", "output file for man page (default: stdout)")
	flag.Parse()

	content, err := cmd.GenerateManPage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating man page: %v\n", err)
		os.Exit(1)
	}

	if outputFile != "" {
		if err := cmd.WriteManPageToFile(outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing man page to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Man page written to %s\n", outputFile)
	} else {
		fmt.Print(content)
	}
}