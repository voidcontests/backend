package ton

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xssnick/tonutils-go/address"
)

func TestFromNano(t *testing.T) {
	tests := []struct {
		name     string
		nano     uint64
		expected string
	}{
		{
			name:     "zero nanotons",
			nano:     0,
			expected: "0",
		},
		{
			name:     "one TON",
			nano:     1_000_000_000,
			expected: "1",
		},
		{
			name:     "fractional TON",
			nano:     500_000_000,
			expected: "0.5",
		},
		{
			name:     "small amount",
			nano:     1,
			expected: "0.000000001",
		},
		{
			name:     "large amount",
			nano:     1_234_567_890_000,
			expected: "1234.56789",
		},
		{
			name:     "10.5 TON",
			nano:     10_500_000_000,
			expected: "10.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromNano(tt.nano)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsTestnet(t *testing.T) {
	t.Run("returns true for testnet client", func(t *testing.T) {
		client := &Client{testnet: true}
		assert.True(t, client.IsTestnet())
	})

	t.Run("returns false for mainnet client", func(t *testing.T) {
		client := &Client{testnet: false}
		assert.False(t, client.IsTestnet())
	})
}

func TestClient_GetAddress(t *testing.T) {
	t.Run("formats address for testnet", func(t *testing.T) {
		addr, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)
		client := &Client{testnet: true}

		result := client.GetAddress(addr)
		assert.NotEmpty(t, result)
		assert.True(t, result[0] == 'k' || result[0] == '0')
	})

	t.Run("formats address for mainnet", func(t *testing.T) {
		addr, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)
		client := &Client{testnet: false}

		result := client.GetAddress(addr)
		assert.NotEmpty(t, result)
		assert.True(t, result[0] == 'E' || result[0] == '0')
	})
}

func TestWallet_Address(t *testing.T) {
	t.Run("returns testnet address when testnet is true", func(t *testing.T) {
		addr, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)
		wallet := &Wallet{
			address: addr,
			testnet: true,
		}

		result := wallet.Address()
		assert.NotNil(t, result)
	})

	t.Run("returns mainnet address when testnet is false", func(t *testing.T) {
		addr, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)
		wallet := &Wallet{
			address: addr,
			testnet: false,
		}

		result := wallet.Address()
		assert.NotNil(t, result)
	})
}

func TestClient_ClearBalanceCache(t *testing.T) {
	t.Run("clears all cached balances", func(t *testing.T) {
		client := &Client{}

		addr1, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)
		addr2, err := address.ParseRawAddr("0:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		assert.NoError(t, err)

		key1 := client.GetAddress(addr1)
		key2 := client.GetAddress(addr2)

		client.balanceCache.Store(key1, &balanceCacheEntry{balance: 1000})
		client.balanceCache.Store(key2, &balanceCacheEntry{balance: 2000})

		_, ok1 := client.balanceCache.Load(key1)
		_, ok2 := client.balanceCache.Load(key2)
		assert.True(t, ok1)
		assert.True(t, ok2)

		client.ClearBalanceCache()

		_, ok1 = client.balanceCache.Load(key1)
		_, ok2 = client.balanceCache.Load(key2)
		assert.False(t, ok1)
		assert.False(t, ok2)
	})

	t.Run("works with empty cache", func(t *testing.T) {
		client := &Client{}

		assert.NotPanics(t, func() {
			client.ClearBalanceCache()
		})
	})
}

func TestClient_InvalidateBalance(t *testing.T) {
	t.Run("invalidates specific address balance", func(t *testing.T) {
		client := &Client{}

		addr1, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)
		addr2, err := address.ParseRawAddr("0:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		assert.NoError(t, err)

		key1 := addr1.String()
		key2 := addr2.String()

		client.balanceCache.Store(key1, &balanceCacheEntry{balance: 1000})
		client.balanceCache.Store(key2, &balanceCacheEntry{balance: 2000})

		client.InvalidateBalance(addr1)

		_, ok1 := client.balanceCache.Load(key1)
		_, ok2 := client.balanceCache.Load(key2)
		assert.False(t, ok1)
		assert.True(t, ok2)
	})

	t.Run("works when address not in cache", func(t *testing.T) {
		client := &Client{}
		addr, err := address.ParseRawAddr("0:8156fbabb2c8c8119dab794ca096f57c3af1775549469f0d1b4e766d8e613c36")
		assert.NoError(t, err)

		assert.NotPanics(t, func() {
			client.InvalidateBalance(addr)
		})
	})
}

func TestConstants(t *testing.T) {
	t.Run("mainnet ID is correct", func(t *testing.T) {
		assert.Equal(t, "-239", MainnetID)
	})

	t.Run("testnet ID is correct", func(t *testing.T) {
		assert.Equal(t, "-3", TestnetID)
	})
}

/*
Integration Tests (require actual blockchain connection):

The following tests would require a connection to TON blockchain (testnet or mainnet)
and should be run as integration tests with proper setup:

1. TestNewClient - requires valid config and network connection
2. TestClient_CreateWallet - requires API client
3. TestClient_WalletWithSeed - requires API client and valid mnemonic
4. TestClient_GetBalance - requires API client and blockchain query
5. TestClient_GetBalanceCached - requires API client, tests caching behavior
6. TestClient_LookupTx - requires API client and actual transactions
7. TestWallet_TransferTo - requires API client, funded wallet, and actual transfer
8. TestClient_RunSmcMethodByID - requires API client and smart contract

Example integration test structure:

func TestIntegration_CreateWallet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	cfg := &config.Ton{
		ConfigURL: "https:		IsTestnet: true,
		Proof: config.TonProof{
			PayloadLifetime:      time.Minute * 5,
			ProofLifetime:        time.Minute * 5,
			PayloadSignatureKey:  "test-key",
		},
	}

	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	wallet, err := client.CreateWallet()
	require.NoError(t, err)
	require.NotNil(t, wallet)
	require.NotNil(t, wallet.Address())
	require.Len(t, wallet.Mnemonic, 24)
}

To run integration tests:
  go test -v ./pkg/ton/... -run Integration
To skip integration tests:
  go test -v ./pkg/ton/... -short
*/
