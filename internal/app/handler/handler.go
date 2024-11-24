package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/cascadecontests/backend/internal/config"
	"github.com/cascadecontests/backend/internal/jwt"
	"github.com/cascadecontests/backend/internal/ton"
	"github.com/labstack/echo/v4"
	"github.com/tonkeeper/tongo/tonconnect"
)

type Handler struct {
	config            *config.Config
	tonconnectMainnet *tonconnect.Server
	tonconnectTestnet *tonconnect.Server
}

func New(config *config.Config, mainnet, testnet *tonconnect.Server) *Handler {
	return &Handler{
		config:            config,
		tonconnectMainnet: mainnet,
		tonconnectTestnet: testnet,
	}
}

func (h *Handler) Healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func (h *Handler) GeneratePayload(c echo.Context) error {
	// 0             8                 16               48
	// | random bits | expiration time | sha2 signature |
	// 0                                        32
	// |                payload                 |

	var err error
	payload, err := h.tonconnectMainnet.GeneratePayload()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}

	return c.JSON(http.StatusInternalServerError, echo.Map{
		"payload": payload,
	})
}

func (h *Handler) CheckProof(c echo.Context) error {
	ctx := c.Request().Context()

	log.Print("call check proof")

	var err error
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		// TODO: Is this really internal server error
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}

	var tp ton.Proof
	err = json.Unmarshal(b, &tp)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid request body",
		})
	}

	var tcs *tonconnect.Server
	switch tp.Network {
	case ton.MainnetID:
		tcs = h.tonconnectMainnet
	case ton.TestnetID:
		tcs = h.tonconnectTestnet
	default:
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid network",
		})
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

	verified, _, err := tcs.CheckProof(ctx, &proof, tcs.CheckPayload, tonconnect.StaticDomain(proof.Proof.Domain))
	if err != nil || !verified {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"message": "tonproof verification failed",
		})
	}

	token, err := jwt.GenerateToken(tp.Address, h.tonconnectMainnet.GetSecret())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
	})
}
