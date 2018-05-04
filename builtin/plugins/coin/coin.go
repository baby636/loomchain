package main

import (
	"errors"

	loom "github.com/loomnetwork/go-loom"
	cointypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
)

var economyKey = []byte("economy")
var decimals = 18

func accountKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("account"), addr.Bytes())
}

type Coin struct {
}

func (c *Coin) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "coin",
		Version: "1.0.0",
	}, nil
}

func (c *Coin) Init(ctx contract.Context, req *cointypes.InitRequest) error {
	for _, acct := range req.Accounts {
		owner := loom.UnmarshalAddressPB(acct.Owner)
		err := ctx.Set(accountKey(owner), acct)
		if err != nil {
			return err
		}
	}

	return nil
}

// ERC20 methods

func (c *Coin) TotalSupply(
	ctx contract.StaticContext,
	req *cointypes.TotalSupplyRequest,
) (*cointypes.TotalSupplyResponse, error) {
	var econ cointypes.Economy
	err := ctx.Get(economyKey, &econ)
	if err != nil {
		return nil, err
	}
	return &cointypes.TotalSupplyResponse{
		TotalSupply: econ.TotalSupply,
	}, nil
}

func (c *Coin) BalanceOf(
	ctx contract.StaticContext,
	req *cointypes.BalanceOfRequest,
) (*cointypes.BalanceOfResponse, error) {
	owner := loom.UnmarshalAddressPB(req.Owner)
	acct, err := loadAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &cointypes.BalanceOfResponse{
		Balance: acct.Balance,
	}, nil
}

func (c *Coin) Transfer(ctx contract.Context, req *cointypes.TransferRequest) error {
	from := ctx.Message().Sender
	to := loom.UnmarshalAddressPB(req.To)

	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}

	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	amount := req.Amount.Value
	fromBalance := fromAccount.Balance.Value
	toBalance := toAccount.Balance.Value

	if fromBalance.Cmp(&amount) < 0 {
		return errors.New("sender balance is too low")
	}

	fromBalance.Sub(&fromBalance, &amount)
	toBalance.Add(&toBalance, &amount)

	fromAccount.Balance.Value = fromBalance
	toAccount.Balance.Value = toBalance
	saveAccount(ctx, fromAccount)
	saveAccount(ctx, toAccount)
	return nil
}

func loadAccount(
	ctx contract.StaticContext,
	owner loom.Address,
) (*cointypes.Account, error) {
	acct := &cointypes.Account{
		Owner: owner.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(0),
		},
	}
	err := ctx.Get(accountKey(owner), acct)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}

	return acct, nil
}

func saveAccount(ctx contract.Context, acct *cointypes.Account) error {
	owner := loom.UnmarshalAddressPB(acct.Owner)
	return ctx.Set(accountKey(owner), acct)
}

var Contract plugin.Contract = contract.MakePluginContract(&Coin{})

func main() {
	plugin.Serve(Contract)
}
