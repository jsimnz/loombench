package main

import (
	"github.com/jsimnz/loombench/types"

	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

func main() {
	plugin.Serve(Contract)
}

type SimpleStore struct{}

var Contract plugin.Contract = contract.MakePluginContract(&SimpleStore{})

func (self *SimpleStore) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "SimpleStore",
		Version: "0.0.1",
	}, nil
}

func (self *SimpleStore) Init(ctx contract.Context, req *plugin.Request) error {
	return nil
}

func (self *SimpleStore) Set(ctx contract.Context, params *types.LoomBenchWriteTx) error {
	return ctx.Set(params.Key, params)
}

func (self *SimpleStore) Get(ctx contract.StaticContext, params *types.LoomBenchReadTx) (*types.LoomBenchResp, error) {
	var result types.LoomBenchResp
	if err := ctx.Get(key, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
