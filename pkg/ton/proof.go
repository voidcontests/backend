package ton

type Proof struct {
	Address string    `json:"address"`
	Network string    `json:"network"`
	Proof   ProofData `json:"proof"`
}

type ProofData struct {
	Timestamp int64  `json:"timestamp"`
	Domain    Domain `json:"domain"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
	StateInit string `json:"state_init"`
}

type Domain struct {
	LengthBytes int    `json:"lengthBytes"`
	Value       string `json:"value"`
}
