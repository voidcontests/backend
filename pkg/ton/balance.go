package ton

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/xssnick/tonutils-go/address"
)

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

func (c *Client) GetBalanceCached(ctx context.Context, address *address.Address) (uint64, error) {
	cacheKey := c.GetAddress(address)
	if cached, ok := c.balanceCache.Load(cacheKey); ok {
		entry := cached.(*balanceCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			slog.Info("cache: returned balance from cache")
			return entry.balance, nil
		}
		c.balanceCache.Delete(cacheKey)
	}

	balance, err := c.GetBalance(ctx, address)
	if err != nil {
		return 0, err
	}

	c.balanceCache.Store(cacheKey, &balanceCacheEntry{
		balance:   balance,
		expiresAt: time.Now().Add(balanceCacheTTL),
	})

	return balance, nil
}

func (c *Client) ClearBalanceCache() {
	c.balanceCache.Range(func(key, value interface{}) bool {
		c.balanceCache.Delete(key)
		return true
	})
}

func (c *Client) InvalidateBalance(address *address.Address) {
	c.balanceCache.Delete(address.String())
}
