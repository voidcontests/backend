package ton

type Proof struct {
	Address string    `json:"address" required:"true"`
	Network string    `json:"network" required:"true"`
	Proof   ProofData `json:"proof" required:"true"`
}

type ProofData struct {
	Timestamp int64  `json:"timestamp" required:"true"`
	Domain    Domain `json:"domain" required:"true"`
	Signature string `json:"signature" required:"true"`
	Payload   string `json:"payload" required:"true"`
	StateInit string `json:"state_init" required:"true"`
}

type Domain struct {
	LengthBytes int    `json:"lengthBytes" required:"true"`
	Value       string `json:"value" required:"true"`
}
