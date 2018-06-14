package loomclient

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	// "github.com/loomnetwork/go-loom/client"
)

type ContractClient struct {
	c         *Contract
	chainID   string
	signer    auth.Signer
	rpcClient *DAppChainRPCClient
}

func NewContractClient(contractAddr, chainID string, signer auth.Signer, rpcClient *DAppChainRPCClient) (*ContractClient, error) {
	contract := &ContractClient{
		chainID:   chainID,
		signer:    signer,
		rpcClient: rpcClient,
	}

	addr, err := contract.resolveAddress(contractAddr)
	if err != nil {
		return nil, err
	}
	contract.c = NewContract(rpcClient, addr.Local)

	return contract, nil
}

func (contract *ContractClient) Call(method string, params proto.Message, result interface{}) error {
	_, err := contract.c.Call(method, params, contract.signer, result)
	return err
}

// func (contract *ContractClient) StaticCall() {
// 	_, err := contract.c.StaticCall(method, params, loom.RootAddress(contract.chainID), result)
// 	return err
// }

func (contract *ContractClient) parseAddress(s string) (loom.Address, error) {
	addr, err := loom.ParseAddress(s)
	if err == nil {
		return addr, nil
	}

	b, err := parseBytes(s)
	if len(b) != 20 {
		return loom.Address{}, loom.ErrInvalidAddress
	}

	return loom.Address{ChainID: contract.chainID, Local: loom.LocalAddress(b)}, nil
}

func (contract *ContractClient) resolveAddress(s string) (loom.Address, error) {
	contractAddr, err := contract.parseAddress(s)
	if err != nil {
		// if address invalid, try to resolve it using registry
		contractAddr, err = contract.rpcClient.Resolve(s)
		if err != nil {
			return loom.Address{}, err
		}
	}

	return contractAddr, nil
}

func parseBytes(s string) ([]byte, error) {
	if strings.HasPrefix(s, "0x") {
		return hex.DecodeString(s[2:])
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		b, err = base64.StdEncoding.DecodeString(s)
	}

	return b, err
}
