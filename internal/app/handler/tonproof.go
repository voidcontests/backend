package handler

import (
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"

	jwtgo "github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/tonkeeper/tongo"
	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/ton"
)

func (h *Handler) GeneratePayload(c echo.Context) error {
	// 0             8                 16               48
	// | random bits | expiration time | sha2 signature |
	// 0                                        32
	// |                payload                 |

	var err error
	payload, err := h.tonconnectMainnet.GeneratePayload()
	if err != nil {
		return err
	}

	slog.Debug("payload generated", slog.String("payload", payload))

	return c.JSON(http.StatusOK, echo.Map{
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
		return err
	}

	var tp ton.Proof
	err = json.Unmarshal(b, &tp)
	if err != nil {
		return Error(http.StatusBadRequest, "invalid request body")
	}

	slog.Debug("tonproof request structure", slog.Any("ton.Proof", tp))

	var tcs *tonconnect.Server
	switch tp.Network {
	case ton.MainnetID:
		tcs = h.tonconnectMainnet
	case ton.TestnetID:
		tcs = h.tonconnectTestnet
	default:
		return Error(http.StatusBadRequest, "invalid network provided")
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

	slog.Debug("proof structure", slog.Any("tonconnect.Proof", proof))

	verified, _, err := tcs.CheckProof(ctx, &proof, tcs.CheckPayload, tonconnect.StaticDomain(proof.Proof.Domain))
	if err != nil || !verified {
		return Error(http.StatusUnauthorized, "tonproof verification failed")
	}

	token, err := jwt.GenerateToken(tp.Address, h.tonconnectMainnet.GetSecret())
	if err != nil {
		return err
	}

	slog.Debug("token generated", slog.String("token", token))

	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
	})
}

func (h *Handler) GetAccount(c echo.Context) error {
	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

	address, err := tongo.ParseAddress(claims.Address)
	if err != nil {
		return Error(http.StatusBadRequest, "can't parse account")
	}

	net := ton.Networks[c.QueryParam("network")]
	if net == nil {
		return Error(http.StatusBadRequest, "undefined network")
	}

	account, err := ton.GetAccountInfo(c.Request().Context(), address.ID, net)
	if err != nil {
		return Error(http.StatusBadRequest, "can't get account info")
	}

	slog.Info("account info", slog.Any("account", account))

	return c.JSON(http.StatusOK, account)
}
