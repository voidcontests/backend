package ton

import (
	"context"
	"fmt"
	"strings"

	"github.com/voidcontests/api/internal/config"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	tonutils "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Client struct {
	api     tonutils.APIClientWrapped
	testnet bool
}

func NewClient(ctx context.Context, c *config.Ton) (*Client, error) {
	client := liteclient.NewConnectionPool()
	err := client.AddConnectionsFromConfigUrl(ctx, c.ConfigURL)
	if err != nil {
		return nil, err
	}

	unsafeAPI := tonutils.NewAPIClient(client, tonutils.ProofCheckPolicyUnsafe)

	block, err := unsafeAPI.GetMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	api := tonutils.NewAPIClient(client, tonutils.ProofCheckPolicySecure).WithRetry()
	api.SetTrustedBlock(block)

	return &Client{
		api:     api,
		testnet: c.IsTestnet,
	}, nil
}

type Wallet struct {
	address  *address.Address
	Mnemonic []string
	Instance *wallet.Wallet
	testnet  bool
}

func (c *Client) CreateWallet() (*Wallet, error) {
	return c.WalletWithSeed(strings.Join(wallet.NewSeed(), " "))
}

func (c *Client) WalletWithSeed(mnemonic string) (*Wallet, error) {
	words := strings.Split(mnemonic, " ")
	w, err := wallet.FromSeedWithOptions(c.api, words, wallet.V4R2)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet from seed: %w", err)
	}

	return &Wallet{
		address:  w.WalletAddress(),
		Mnemonic: words,
		Instance: w,
		testnet:  c.testnet,
	}, nil
}

func (c *Client) GetBalance(ctx context.Context, address *address.Address) (uint64, error) {
	block, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get masterchain info: %w", err)
	}

	account, err := c.api.GetAccount(ctx, block, address)
	if err != nil {
		return 0, fmt.Errorf("failed to get account: %w", err)
	}

	if !account.IsActive {
		return 0, nil
	}

	return account.State.Balance.Nano().Uint64(), nil
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

func FromNano(nano uint64) string {
	return tlb.FromNanoTONU(nano).String()
}

func (c *Client) LookupTx(ctx context.Context, from *address.Address, to *address.Address, amount tlb.Coins) (string, bool) {
	block, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return "", false
	}

	account, err := c.api.GetAccount(ctx, block, to)
	if err != nil {
		return "", false
	}

	if !account.IsActive {
		return "", false
	}

	txs, err := c.api.ListTransactions(ctx, to, 100, account.LastTxLT, account.LastTxHash)
	if err != nil {
		return "", false
	}

	for _, tx := range txs {
		if tx.IO.In == nil || tx.IO.In.MsgType != tlb.MsgTypeInternal {
			continue
		}

		inmsg := tx.IO.In.AsInternal()
		if inmsg == nil {
			continue
		}

		if inmsg.SrcAddr.Equals(from) {
			// checks if in tx transferred at least `amount`
			if inmsg.Amount.Nano().Cmp(amount.Nano()) >= 0 {
				return fmt.Sprintf("%x", tx.Hash), true
			}
		}
	}

	return "", false
}

func (w *Wallet) Address() *address.Address {
	return w.address.Testnet(w.testnet)
}

func (c *Client) GetAddressString(addr *address.Address) string {
	return addr.Testnet(c.testnet).String()
}
