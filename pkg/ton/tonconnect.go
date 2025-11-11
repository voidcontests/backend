package ton

import (
	"github.com/tonkeeper/tongo/liteapi"
)

const (
	MainnetID = "-239"
	TestnetID = "-3"
)

func Mainnet() *liteapi.Client {
	client, _ := liteapi.NewClientWithDefaultMainnet()
	return client
}

func Testnet() *liteapi.Client {
	client, _ := liteapi.NewClientWithDefaultTestnet()
	return client
}
