package main

import (
	"github.com/spf13/cobra"
	"kryptcoin/fs"
	"log"
)

const flagDataDir = "data_dir"
const flagIP = "ip"
const flagPort = "port"
const flagMiner = "miner"
const flagKeystoreFile = "keystore"
const flagBootstrapAcct = "bootstrap_account"
const flagBootstrapIp = "bootstrap_ip"
const flagBootstrapPort = "bootstrap_port"

func main() {
	tbbCmd := &cobra.Command{
		Use:   "tbb",
		Short: "The One Piece Berries Blockchain CLI",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	tbbCmd.AddCommand(getBalancesCmd())
	tbbCmd.AddCommand(getRunCmd())
	tbbCmd.AddCommand(walletCmd())

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

func addKeystoreFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagKeystoreFile, "", "Absolute path to the encrypted keystore file.")
	cmd.MarkFlagRequired(flagKeystoreFile)
}
