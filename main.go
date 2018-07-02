package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
)

type DiceContract struct {
}

func (c *DiceContract) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "DiceContract",
		Version: "1.0.0",
	}, nil
}

var Contract = contractpb.MakePluginContract(&DiceContract{})

func main() {
	plugin.Serve(Contract)
}
