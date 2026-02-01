package ton

import (
	"context"

	"github.com/tonkeeper/tongo/tlb"
	"github.com/tonkeeper/tongo/ton"
	"github.com/xssnick/tonutils-go/address"
)

// RunSmcMethodByID is an implementation of `abi.Executor` interface for `tonconnect.NewTonConnect` function
func (c *Client) RunSmcMethodByID(ctx context.Context, accountID ton.AccountID, methodID int, params tlb.VmStack) (uint32, tlb.VmStack, error) {
	rawAddr := accountID.ToRaw()
	addr, err := address.ParseAddr(rawAddr)
	if err != nil {
		return 0, tlb.VmStack{}, err
	}

	block, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, tlb.VmStack{}, err
	}

	var tuParams []interface{}

	methodName := ""
	_, err = c.api.RunGetMethod(ctx, block, addr, methodName, tuParams...)
	if err != nil {
		return 0, tlb.VmStack{}, err
	}

	return 0, tlb.VmStack{}, nil
}
