package main

import (
	"encoding/json"
	"log"
	"strings"

	"math/rand"

	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
	"github.com/void-main/loomdice-core/txmsg"
)

type DiceContract struct {
}

type UserState struct {
	ChipCount int32   `json:"chips"`
	WinCount  int32   `json:"win"`
	LoseCount int32   `json:"lose"`
	History   []int32 `json:"history"`
}

func NewUserState() UserState {
	return UserState{
		ChipCount: 100,
		WinCount:  0,
		LoseCount: 0,
		History:   make([]int32, 0),
	}
}

func (c *DiceContract) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "loomdicecore",
		Version: "1.0.0",
	}, nil
}

func (dc *DiceContract) ownerKey(owner string) []byte {
	return []byte("owner:" + owner)
}

func (dc *DiceContract) Init(ctx contract.Context, req *plugin.Request) error {
	return nil
}

func (dc *DiceContract) CreateAccount(ctx contract.Context, accTx *txmsg.LDCreateAccountTx) error {
	owner := strings.TrimSpace(accTx.Owner)
	if ctx.Has(dc.ownerKey(owner)) {
		return errors.New("Owner already exists")
	}
	addr := []byte(ctx.Message().Sender.Local)

	initState, err := json.Marshal(NewUserState())
	if err != nil {
		return errors.Wrap(err, "Error marshalling state")
	}
	ctx.Logger().Info("Owner: ", owner)
	ctx.Logger().Info("Init state", string(initState))
	state := txmsg.LDAppState{
		State: initState,
	}
	if err := ctx.Set(dc.ownerKey(owner), &state); err != nil {
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

func (dc *DiceContract) GetState(ctx contract.StaticContext, params *txmsg.LDStateQueryParams) (*txmsg.LDStateQueryResult, error) {
	ctx.Logger().Info("Get State", "owner", params.Owner)
	if ctx.Has(dc.ownerKey(params.Owner)) {
		var curState txmsg.LDAppState
		if err := ctx.Get(dc.ownerKey(params.Owner), &curState); err != nil {
			return nil, err
		}
		return &txmsg.LDStateQueryResult{State: curState.State}, nil
	}
	return &txmsg.LDStateQueryResult{}, nil
}

func (dc *DiceContract) GetChipCount(ctx contract.StaticContext, params *txmsg.LDChipQueryParams) (*txmsg.LDChipQueryResult, error) {
	ctx.Logger().Info("Get Chip Count", "owner", params.Owner)
	if ctx.Has(dc.ownerKey(params.Owner)) {
		var curState txmsg.LDAppState
		if err := ctx.Get(dc.ownerKey(params.Owner), &curState); err != nil {
			return nil, err
		}

		var state UserState
		err := json.Unmarshal(curState.GetState(), &state)
		if err != nil {
			return nil, err
		}

		return &txmsg.LDChipQueryResult{
			Amount: state.ChipCount,
		}, nil
	}
	return nil, errors.Errorf("unknown account %v", params.Owner)
}

func (dc *DiceContract) Roll(ctx contract.Context, params *txmsg.LDRollQueryParams) (*txmsg.LDRollQueryResult, error) {
	owner := strings.TrimSpace(params.Owner)
	ctx.Logger().Info("Roll Params", "owner", owner, "betBig", params.BetBig, "amount", params.Amount)
	if ctx.Has(dc.ownerKey(owner)) {
		var curState txmsg.LDAppState
		if err := ctx.Get(dc.ownerKey(owner), &curState); err != nil {
			return nil, err
		}

		if ok, _ := ctx.HasPermission([]byte(owner), []string{"owner"}); !ok {
			return nil, errors.New("Owner unverified")
		}

		var state UserState
		err := json.Unmarshal(curState.GetState(), &state)
		if err != nil {
			return nil, err
		}

		if params.Amount > state.ChipCount {
			return nil, errors.Errorf("not enough chips. current have: %v, bet %v", state.ChipCount, params.Amount)
		}

		number := rand.Intn(5) + 1

		result := txmsg.LDRollQueryResult{}
		if (params.BetBig && number >= 4) || (!params.BetBig && number <= 3) { // you win!
			// setup result
			result.Point = int32(number)
			result.Win = true
			result.Amount = state.ChipCount + params.Amount

			// update state
			state.ChipCount = result.Amount
			state.WinCount += 1
			state.History = append(state.History, params.Amount)
		} else { // sorry, you lose
			// setup result
			result.Point = int32(number)
			result.Win = false
			result.Amount = state.ChipCount - params.Amount

			// update state
			state.ChipCount = result.Amount
			state.LoseCount += 1
			state.History = append(state.History, -params.Amount)
		}

		s, err := json.Marshal(state)
		if err != nil {
			return nil, err
		}
		curState = txmsg.LDAppState{
			State: s,
		}

		if err := ctx.Set(dc.ownerKey(owner), &curState); err != nil {
			return nil, errors.Wrap(err, "Error marshaling state node")
		}

		return &result, nil
	}
	return nil, errors.Errorf("unknown account %v, create it first", params.Owner)
}

var Contract = contract.MakePluginContract(&DiceContract{})

func main() {
	plugin.Serve(Contract)
}
