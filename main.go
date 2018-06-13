package main

import (
	// "flag"
	"fmt"
	"os"
	// "testing"

	"github.com/spf13/cobra"
)

// var txFlags struct {
// 	WriteURI     string
// 	ReadURI      string
// 	ContractAddr string
// 	ChainID      string
// 	PrivFile     string
// }

// func ContractCallCommand() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "call",
// 		Short: "call a contract method",
// 	}
// 	pflags := cmd.PersistentFlags()
// 	pflags.StringVarP(&txFlags.WriteURI, "write", "w", "http://localhost:46658/rpc", "URI for sending txs")
// 	pflags.StringVarP(&txFlags.ReadURI, "read", "r", "http://localhost:46658/query", "URI for quering app state")
// 	pflags.StringVarP(&txFlags.ContractAddr, "contract", "", "", "contract address")
// 	pflags.StringVarP(&txFlags.ChainID, "chain", "", "default", "chain ID")
// 	pflags.StringVarP(&txFlags.PrivFile, "private-key", "p", "", "private key file")
// 	return cmd
// }

var (
	config string
)

func init() {
	// cobra.OnInitialize(initConfig)
	// flag.StringVar(&config, "config", "", "config file")
	// flag.Parse()
	// fmt.Println("Config From Init: ", config)
	fmt.Println("main")
}

var rootCmd = &cobra.Command{
	Use:   "loombench",
	Short: "LoomBench is a benchmarking utility for the Loom Network DAppChain platform",
	Long: `An easy to use utility to stress test a deployed Loom DAppChain
                in any kind of environment.
                Complete documentation is available at http://jsimnz.github.cio/loombench`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Config: '%s'\n", config)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
	// fmt.Println("Run: go test -bench=.")
}
