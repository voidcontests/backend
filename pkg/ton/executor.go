package ton

import (
	"context"

	"github.com/tonkeeper/tongo/abi"
	"github.com/tonkeeper/tongo/tlb"
	"github.com/tonkeeper/tongo/ton"
	"github.com/xssnick/tonutils-go/address"
	tonutils "github.com/xssnick/tonutils-go/ton"
)

// TODO: document this package, because liteapi is shit

type ExecutorAdapter struct {
	api tonutils.APIClientWrapped
}

func NewExecutorAdapter(api tonutils.APIClientWrapped) abi.Executor {
	return &ExecutorAdapter{api: api}
}

func (e *ExecutorAdapter) RunSmcMethodByID(ctx context.Context, accountID ton.AccountID, methodID int, params tlb.VmStack) (uint32, tlb.VmStack, error) {
	rawAddr := accountID.ToRaw()
	addr, err := address.ParseAddr(rawAddr)
	if err != nil {
		return 0, tlb.VmStack{}, err
	}

	block, err := e.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, tlb.VmStack{}, err
	}

	var tuParams []interface{}

	methodName := ""
	_, err = e.api.RunGetMethod(ctx, block, addr, methodName, tuParams...)
	if err != nil {
		return 0, tlb.VmStack{}, err
	}

	return 0, tlb.VmStack{}, nil
}
