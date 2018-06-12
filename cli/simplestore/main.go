package main

import (
	"encoding/base64"
	// "errors"go
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jsimnz/loombench/types"

	"github.com/loomnetwork/go-loom/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ed25519"
)

func getKeygenCmd() *cobra.Command {
	var privFile string
	keygenCmd := &cobra.Command{
		Use:           "genkey",
		Short:         "generate a public and private key pair",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				return errors.Wrapf(err, "Error generating key pair")
			}
			data := base64.StdEncoding.EncodeToString(priv)
			if err := ioutil.WriteFile(privFile, []byte(data), 0664); err != nil {
				return errors.Wrapf(err, "Unable to write private key")
			}
			fmt.Printf("written private key file '%s'\n", privFile)
			return nil
		},
	}
	keygenCmd.Flags().StringVarP(&privFile, "key", "k", "key", "private key file")
	return keygenCmd
}

func main() {
	var key string
	var value string
	defaultContract := "SimpleStore"

	rootCmd := &cobra.Command{
		Use:   "simplestore",
		Short: "SimpleStore",
	}

	callCmd := cli.ContractCallCommand()
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(getKeygenCmd())

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "set the state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" || value == "" {
				return errors.New("Missing key or value args")
			}

			msg := &types.SetParams{
				Key:   []byte(key),
				Value: []byte(value),
			}

			err := cli.CallContract(defaultContract, "Set", msg, nil)
			if err != nil {
				return err
			}

			fmt.Printf("Set key %s to value %s\n", key, value)
			return nil
		},
	}
	setCmd.Flags().StringVarP(&key, "key", "k", "", "key to set")
	setCmd.MarkFlagRequired("key")
	setCmd.Flags().StringVarP(&value, "value", "v", "", "value to set")
	setCmd.MarkFlagRequired("value")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "get the state",
		RunE: func(cmd *cobra.Command, args []string) error {

			msg := &types.QueryParams{
				Key: []byte(key),
			}

			var result types.QueryResult
			err := cli.StaticCallContract(defaultContract, "Get", msg, &result)
			if err != nil {
				return err
			}

			fmt.Printf("Got value %s for key %s\n", string(result.Value), string(result.Key))

			return nil
		},
	}

	getCmd.Flags().StringVarP(&key, "key", "k", "", "key to set")
	getCmd.MarkFlagRequired("key")

	callCmd.AddCommand(setCmd)
	callCmd.AddCommand(getCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

}
