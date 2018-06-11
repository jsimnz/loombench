package main

import (
	"fmt"
	"os"
	"errors"

	"github.com/jsimnz/loombench/types"

	"github.com/spf13/cobra"
	"github.com/loomnetwork/go-loom/cli"
)

func main() {
	var key string
	var value string 
	defaultContract := "Simplestore"

	rootCmd := &cobra.Command{
		Use: "simplestore",
		Short: "SimpleStore",
	}

	callCmd := cli.ContractCallCommand()
	rootCmd.AddCommand(callCmd)

	setCmd := &cobra.Command{
		Use: "set",
		Short: "set the state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" || value == "" {
				return errors.New("Missing key or value args")
			}

			msg := &types.SetParams{
				Key: []byte(key),
				Value: []byte(value),
			}

			err := cli.CallContract(defaultContract, "Set", msg, nil)
			if err != nil {
				return err
			}

			fmt.Printf("Set key %s to value %s", key, value)
			return nil
		},
	}
	setCmd.Flags().StringVarP(&key, "key", "k", "", "key to set")
	setCmd.Flags().StringVarP(&value, "value", "v", "", "value to set")

	callCmd.AddCommand(setCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}