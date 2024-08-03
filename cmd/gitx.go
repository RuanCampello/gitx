package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var gitxCmd = &cobra.Command{
	Use:   "gitx",
	Short: "Get your github repos state instantly",
}

func Execute() {
	if err := gitxCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
