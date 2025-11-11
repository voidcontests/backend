package ton

import (
	"context"
	"fmt"
	"strings"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Wallet struct {
	address  *address.Address
	Mnemonic []string
	Instance *wallet.Wallet
	testnet  bool
}

func (w *Wallet) TransferTo(ctx context.Context, recipient *address.Address, amount tlb.Coins, comments ...string) (tx string, err error) {
	var body *cell.Cell

	if len(comments) > 0 {
		comment := strings.Join(comments, ", ")
		body, err = wallet.CreateCommentCell(comment)
		if err != nil {
			return "", fmt.Errorf("failed to create comment cell: %w", err)
		}
	}

	transaction, _, err := w.Instance.SendWaitTransaction(ctx, &wallet.Message{
		Mode: wallet.PayGasSeparately + wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      false,
			DstAddr:     recipient,
			Amount:      amount,
			Body:        body,
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return fmt.Sprintf("%x", transaction.Hash), nil
}

func (w *Wallet) Address() *address.Address {
	return w.address.Testnet(w.testnet)
}
