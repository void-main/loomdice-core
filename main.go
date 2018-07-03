package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
	"github.com/void-main/loomdice-core/txmsg"
)

type DiceContract struct {
}

type UserState struct {
	ChipCount int32 `json:"chips"`
	WinCount  int32 `json:"win"`
	LoseCount int32 `json:"lose"`
	History   []int `json:"history"`
}

func NewUserState() UserState {
	return UserState{
		ChipCount: 0,
		WinCount:  0,
		LoseCount: 0,
		History:   make([]int, 0),
	}
}

func (c *DiceContract) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "loomdicecore",
		Version: "1.0.0",
	}, nil
}

func (c *DiceContract) ownerKey(owner string) []byte {
	return []byte("owner:" + owner)
}

func (e *DiceContract) Init(ctx contract.Context, req *plugin.Request) error {
	return nil
}

func (e *DiceContract) CreateAccount(ctx contract.Context, accTx *txmsg.LDCreateAccountTx) error {
	owner := strings.TrimSpace(accTx.Owner)
	if ctx.Has(e.ownerKey(owner)) {
		return errors.New("Owner already exists")
	}
	addr := []byte(ctx.Message().Sender.Local)

	st := NewUserState()
	ctx.Logger().Info("Before marshal", "state", st)
	initState, err := json.Marshal(st)
	if err != nil {
		return errors.Wrap(err, "Error marshalling state")
	}
	ctx.Logger().Info("Owner: ", owner)
	ctx.Logger().Info("Init state", initState)
	state := txmsg.LDAppState{
		Address: addr,
		State:   initState,
	}
	if err := ctx.Set(e.ownerKey(owner), &state); err != nil {
		return errors.Wrap(err, "Error setting state")
	}
	ctx.GrantPermission([]byte(owner), []string{"owner"})
	ctx.Logger().Info("Created account", "owner", owner, "address", addr)
	emitMsg := struct {
		Owner  string
		Method string
		Addr   []byte
	}{owner, "createacct", addr}
	emitMsgJSON, err := json.Marshal(emitMsg)
	if err != nil {
		log.Println("Error marshalling emit message")
	}
	ctx.EmitTopics(emitMsgJSON, "loomdice:createaccount")
	return nil
}

func (e *DiceContract) GetState(ctx contract.StaticContext, params *txmsg.LDStateQueryParams) (*txmsg.LDStateQueryResult, error) {
	ctx.Logger().Info("Get State", "owner", params.Owner)
	if ctx.Has(e.ownerKey(params.Owner)) {
		var curState txmsg.LDAppState
		if err := ctx.Get(e.ownerKey(params.Owner), &curState); err != nil {
			return nil, err
		}
		return &txmsg.LDStateQueryResult{State: curState.State}, nil
	}
	return &txmsg.LDStateQueryResult{}, nil
}

var Contract = contract.MakePluginContract(&DiceContract{})

func main() {
	plugin.Serve(Contract)
}
