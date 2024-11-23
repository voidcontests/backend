package ton

import "github.com/tonkeeper/tongo/liteapi"

const (
	MainnetID = "-239"
	TestnetID = "-3"
)

var Networks = make(map[string]*liteapi.Client, 0)

func Mainnet() *liteapi.Client {
	return Networks[MainnetID]
}

func Testnet() *liteapi.Client {
	return Networks[TestnetID]
}
