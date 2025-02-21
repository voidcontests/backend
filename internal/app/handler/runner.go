package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
)

func (h *Handler) Run(c echo.Context) error {
	body := `{"code": "#include <stdio.h>\nint main() { return 69; }"}`
	req, err := http.NewRequest("POST", "http://localhost:2111/run", strings.NewReader(body))
	if err != nil {
		slog.Error("something went wrong 1", sl.Err(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("something went wrong 2", sl.Err(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("something went wrong 3", slog.Int("status", resp.StatusCode))
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data struct {
		Status int    `json:"status"`
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("something went wrong 4", sl.Err(err))
		return err
	}

	err = json.Unmarshal(response, &data)
	if err != nil {
		slog.Error("failed to unmarshal response", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusOK, data)
}
