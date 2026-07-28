package main

import (
	"fmt"
	"os"

	"github.com/TykTechnologies/gromit/dhi"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: customization-equal <left.yaml> <right.yaml>")
		os.Exit(2)
	}

	left, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer left.Close()

	right, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer right.Close()

	equal, err := dhi.EqualCustomizations(left, right)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if equal {
		fmt.Println("equal")
		return
	}
	fmt.Println("different")
}
