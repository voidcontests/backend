package runner

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/voidcontests/backend/internal/app/handler/dto/request"
)

const BASEPATH = "http://localhost:21003"

type ExecutionRequest struct {
	Code     string `json:"code" required:"true"`
	Language string `json:"language" required:"true"`
	Input    string `json:"input"`
}

type ExecutionResponse struct {
	Status int    `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type TestingRequest struct {
	Code        string       `json:"code"`
	Language    string       `json:"language" required:"true"`
	TimeLimitMS int          `json:"time_limit_ms"`
	TCs         []request.TC `json:"tcs"`
}

type TestingResponse struct {
	Verdict    string     `json:"verdict"`
	Passed     int        `json:"passed"`
	Total      int        `json:"total"`
	Stderr     string     `json:"stderr"`
	FailedTest FailedTest `json:"failed_test,omitempty"`
}

type FailedTest struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	ActualOutput   string `json:"actual_output"`
}

func ExecuteTesting(code, language string, timeLimitMS int, tcs []request.TC) (*TestingResponse, error) {
	body := TestingRequest{
		Code:        code,
		Language:    language,
		TimeLimitMS: timeLimitMS,
		TCs:         tcs,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", BASEPATH+"/test", bytes.NewBuffer(raw))
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

	var tr TestingResponse
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &tr)
	if err != nil {
		return nil, err
	}

	return &tr, nil
}
