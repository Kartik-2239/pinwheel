package main

import (
	"fmt"
	"os"

	"github.com/Kartik-2239/pinwheel/internal/cli"
)

func main() {
	if err := cli.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
