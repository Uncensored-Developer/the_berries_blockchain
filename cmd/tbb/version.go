package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

const (
	Major  = "0"
	Minor  = "1"
	Fix    = "0"
	Verbal = "Add Txns and List Balances"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Describes version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s.%s.%s %s\n", Major, Minor, Fix, Verbal)
	},
}
