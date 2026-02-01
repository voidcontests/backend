package ton

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/api/internal/config"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	tonutils "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	MainnetID = "-239"
	TestnetID = "-3"
)

type Client struct {
	api          tonutils.APIClientWrapped
	testnet      bool
	TonConnect   *tonconnect.Server
	balanceCache sync.Map // map[string]*balanceCacheEntry, key is address string
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

	tc := &Client{
		api:     api,
		testnet: c.IsTestnet,
	}

	payloadLifetime := int64(c.Proof.PayloadLifetime.Seconds())
	proofLifetime := int64(c.Proof.ProofLifetime.Seconds())

	tcserver, err := tonconnect.NewTonConnect(
		tc,
		c.Proof.PayloadSignatureKey,
		tonconnect.WithLifeTimePayload(payloadLifetime),
		tonconnect.WithLifeTimeProof(proofLifetime),
	)
	if err != nil {
		return nil, fmt.Errorf("tonconnect: can't initialize: %w", err)
	}

	tc.TonConnect = tcserver

	return tc, nil
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

func (c *Client) IsTestnet() bool {
	return c.testnet
}

func (c *Client) API() tonutils.APIClientWrapped {
	return c.api
}
