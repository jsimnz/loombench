package main

import (
	"encoding/base64"
	// "errors"go
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/jsimnz/loombench/loomclient"
	"github.com/jsimnz/loombench/types"

	"github.com/loomnetwork/go-loom/auth"
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

var txFlags struct {
	WriteURI     string
	ReadURI      string
	ContractAddr string
	ChainID      string
	PrivFile     string
}

func init() {

}

func main() {
	var key string
	var value string
	// defaultContract := "SimpleStore"

	rootCmd := &cobra.Command{
		Use:   "simplestore",
		Short: "SimpleStore",
	}

	// callCmd := cli.ContractCallCommand()
	callCmd := &cobra.Command{
		Use:   "call",
		Short: "call a contract method",
	}
	pflags := callCmd.PersistentFlags()
	pflags.StringVarP(&txFlags.WriteURI, "write", "w", "http://localhost:46658/rpc", "URI for sending txs")
	pflags.StringVarP(&txFlags.ReadURI, "read", "r", "http://localhost:46658/query", "URI for quering app state")
	pflags.StringVarP(&txFlags.ContractAddr, "contract", "", "SimpleStore", "contract address")
	pflags.StringVarP(&txFlags.ChainID, "chain", "", "default", "chain ID")
	pflags.StringVarP(&txFlags.PrivFile, "private-key", "p", "", "private key file")

	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(getKeygenCmd())

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "set the state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" || value == "" {
				return errors.New("Missing key or value args")
			}

			msg := &types.LoomBenchWriteTx{
				Key: []byte(key),
				Val: []byte(value),
			}

			c, err := newLoomClient()
			if err != nil {
				return err
			}

			err = c.Call("Set", msg, nil)
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

	// getCmd := &cobra.Command{
	// 	Use:   "get",
	// 	Short: "get the state",
	// 	RunE: func(cmd *cobra.Command, args []string) error {

	// 		msg := &types.QueryParams{
	// 			Key: []byte(key),
	// 		}

	// 		var result types.LoomBenchResp
	// 		err := cli.StaticCallContract(defaultContract, "Get", msg, &result)
	// 		_ = cli.StaticCallContract(defaultContract, "Get", msg, nil)
	// 		if err != nil {
	// 			return err
	// 		}

	// 		fmt.Printf("Got value %s for key %s\n", string(result.Val), key)

	// 		return nil
	// 	},
	// }

	// getCmd.Flags().StringVarP(&key, "key", "k", "", "key to set")
	// getCmd.MarkFlagRequired("key")

	callCmd.AddCommand(setCmd)
	// callCmd.AddCommand(getCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

}

func newLoomClient() (*loomclient.ContractClient, error) {
	// create signer
	// var signer *auth.Signer
	var privKey []byte
	var err error
	if txFlags.PrivFile == "genkey" {
		_, privKey, err = ed25519.GenerateKey(nil)
	} else {
		privKeyB64, err := ioutil.ReadFile(txFlags.PrivFile)
		if err != nil {
			return nil, err
		}

		privKey, err = base64.StdEncoding.DecodeString(string(privKeyB64))
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	signer := auth.NewEd25519Signer(privKey)

	httpclient := &http.Client{}
	rpcClient := loomclient.NewDAppChainRPCClient(httpclient, txFlags.ChainID, txFlags.WriteURI, txFlags.ReadURI)
	client, err := loomclient.NewContractClient(txFlags.ContractAddr, txFlags.ChainID, signer, rpcClient)

	return client, err
}
