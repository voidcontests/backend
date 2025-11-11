package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/pkg/ton"
	"github.com/voidcontests/api/pkg/validate"
	"github.com/xssnick/tonutils-go/address"
)

// TODO: Wrap errors

func (h *Handler) GeneratePayload(c echo.Context) error {
	// 0             8                 16               48
	// | random bits | expiration time | sha2 signature |
	// 0                                        32
	// |                payload                 |

	payload, err := h.tcs.GeneratePayload()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"payload": payload,
	})
}

func (h *Handler) CheckProof(c echo.Context) error {
	ctx := c.Request().Context()

	claims, ok := ExtractClaims(c)
	if !ok {
		return Error(http.StatusUnauthorized, "user not authenticated")
	}

	var tp request.TonProof
	if err := validate.Bind(c, &tp); err != nil {
		return Error(http.StatusBadRequest, "invalid request body")
	}

	expectedNetwork := ton.MainnetID
	if h.config.Ton.IsTestnet {
		expectedNetwork = ton.TestnetID
	}
	if tp.Network != expectedNetwork {
		return Error(http.StatusBadRequest, "network mismatch")
	}

	proof := tonconnect.Proof{
		Address: tp.Address,
		Proof: tonconnect.ProofData{
			Timestamp: tp.Proof.Timestamp,
			Domain:    tp.Proof.Domain.Value,
			Signature: tp.Proof.Signature,
			Payload:   tp.Proof.Payload,
			StateInit: tp.Proof.StateInit,
		},
	}

	verified, _, err := h.tcs.CheckProof(ctx, &proof, h.tcs.CheckPayload, tonconnect.StaticDomain(tp.Proof.Domain.Value))
	if err != nil || !verified {
		return Error(http.StatusUnauthorized, "tonproof verification failed")
	}

	addr, err := address.ParseRawAddr(tp.Address)
	if err != nil {
		return err
	}

	addrstr := addr.Testnet(h.config.Ton.IsTestnet).String()

	_, err = h.repo.User.UpdateUser(ctx, claims.UserID, models.UpdateUserParams{
		Address: &addrstr,
	})
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}
