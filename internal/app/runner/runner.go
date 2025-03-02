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
	Code  string `json:"code" required:"true"`
	Input string `json:"input"`
}

type ExecutionResponse struct {
	Status int    `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type TestingRequest struct {
	Code string       `json:"code"`
	TCs  []request.TC `json:"tcs"`
}

type TestingResponse struct {
	Verdict string `json:"verdict"`
	Passed  int    `json:"passed"`
	Total   int    `json:"total"`
}

func ExecuteTesting(code string, tcs []request.TC) (*TestingResponse, error) {
	body := TestingRequest{
		Code: code,
		TCs:  tcs,
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

func ExecuteWithInput(code string, input string) (*ExecutionResponse, error) {
	body := ExecutionRequest{
		Code:  code,
		Input: input,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", BASEPATH+"/run", bytes.NewBuffer(raw))
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

	var er ExecutionResponse
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &er)
	if err != nil {
		return nil, err
	}

	return &er, nil
}
