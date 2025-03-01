package runner

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type Result struct {
	Status int    `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type request struct {
	Code  string `json:"code" required:"true"`
	Input string `json:"input"`
}

func ExecuteWithInput(code string, input string) (*Result, error) {
	body := request{
		Code:  code,
		Input: input,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "http://localhost:2111/run", bytes.NewBuffer(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var r Result
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
