package main

import (
	"github.com/spf13/cobra"
	"log"
)

func main() {
	tbbCmd := &cobra.Command{
		Use:   "tbb",
		Short: "The Berries Blockchain CLI",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	tbbCmd.AddCommand(versionCmd)
	tbbCmd.AddCommand(getBalancesCmd())

	err := tbbCmd.Execute()
	if err != nil {
		log.Fatalf("Could not execute command: %v\n", err)
	}
}
