package ton

type Proof struct {
	Address string `json:"address"`
	Network string `json:"network"`
	Proof   struct {
		Timestamp int64 `json:"timestamp"`
		Domain    struct {
			Value string `json:"value"`
		} `json:"domain"`
		Signature string `json:"signature"`
		Payload   string `json:"payload"`
		StateInit string `json:"state_init"`
	} `json:"proof"`
}
