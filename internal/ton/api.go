package ton

import (
	"context"

	"github.com/tonkeeper/tongo"
	"github.com/tonkeeper/tongo/liteapi"
)

func GetAccountInfo(ctx context.Context, address tongo.AccountID, net *liteapi.Client) (*Account, error) {
	account, err := net.GetAccountState(ctx, address)
	if err != nil {
		return nil, err
	}
	accountInfo := Account{
		Balance: int64(account.Account.Account.Storage.Balance.Grams),
		Status:  string(account.Account.Status()),
	}
	accountInfo.Address.Raw = address.ToRaw()
	accountInfo.Address.Bounceable = address.ToHuman(true, false)
	accountInfo.Address.NonBounceable = address.ToHuman(false, false)

	return &accountInfo, nil
}
