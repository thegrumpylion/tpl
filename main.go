package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "tpl",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: %s", err)
	}
}
