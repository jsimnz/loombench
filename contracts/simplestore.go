package main

import (
	"github.com/jsimnz/loombench/types"

	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

func main() {
	plugin.Serve(Contract)
}

type SimpleStore struct {}

var Contract plugin.Contract = contract.MakePluginContract(&SimpleStore{})

func (self *SimpleStore) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name: "SimpleStore",
		Version: "0.0.1",
	}, nil
}

func (self *SimpleStore) Set(ctx contract.Context, params *types.SetParams) error {
	return ctx.Set(params.Key, params)
}

func (self *SimpleStore) SetEcho(ctx contract.Context, params *types.SetParams) (*types.QueryResult, error) {
	if err := ctx.Set(params.Key, params); err != nil {
		return nil, err
	}
	return self.doGet(ctx, params.Key)
}

func (self *SimpleStore) Get(ctx contract.StaticContext, params *types.QueryParams) (*types.QueryResult, error) {
	return self.doGet(ctx, params.Key)
}

func (self *SimpleStore) doGet(ctx contract.StaticContext, key []byte) (*types.QueryResult, error) {
	var result types.QueryResult
	if err := ctx.Get(key, &result); err != nil {
		return nil, err
	}

	return &result, nil
}