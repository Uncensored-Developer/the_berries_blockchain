package main

import (
	"github.com/spf13/cobra"
	"kryptcoin/fs"
	"log"
)

const flagDataDir = "data_dir"

func main() {
	tbbCmd := &cobra.Command{
		Use:   "tbb",
		Short: "The Berries Blockchain CLI",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	tbbCmd.AddCommand(versionCmd)
	tbbCmd.AddCommand(getBalancesCmd())
	tbbCmd.AddCommand(getRunCmd())
	tbbCmd.AddCommand(getMigrateCmd())

	err := tbbCmd.Execute()
	if err != nil {
		log.Fatalf("Could not execute command: %v\n", err)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to the node data dir where the DB is stored.")
	cmd.MarkFlagRequired(flagDataDir)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)
	return fs.ExpandPath(dataDir)
}
