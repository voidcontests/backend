package ton

import "github.com/xssnick/tonutils-go/address"

func (c *Client) GetAddress(addr *address.Address) string {
	return addr.Testnet(c.testnet).String()
}
