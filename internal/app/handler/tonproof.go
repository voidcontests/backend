package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	jwtgo "github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/internal/ton"
	"github.com/voidcontests/backend/pkg/requestid"
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

	return c.JSON(http.StatusOK, map[string]any{
		"payload": payload,
	})
}

func (h *Handler) CheckProof(c echo.Context) error {
	ctx := c.Request().Context()

	var err error
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return Error(http.StatusBadRequest, "can't read request body")
	}

	var tp ton.Proof
	err = json.Unmarshal(b, &tp)
	if err != nil {
		return Error(http.StatusBadRequest, "invalid request body")
	}

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

	verified, _, err := tcs.CheckProof(ctx, &proof, tcs.CheckPayload, tonconnect.StaticDomain(proof.Proof.Domain))
	if err != nil || !verified {
		return Error(http.StatusUnauthorized, "tonproof verification failed")
	}

	user, err := h.repo.User.GetByAddress(c.Request().Context(), tp.Address)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		user, err = h.repo.User.Create(c.Request().Context(), tp.Address)
		if err != nil {
			slog.Error("can't create user instance in database", sl.Err(err))
			return err
		}
	} else if err != nil {
		slog.Error("can't get user by address", sl.Err(err))
		return err
	}

	token, err := jwt.GenerateToken(tp.Address, user.ID, h.tonconnectMainnet.GetSecret())
	if err != nil {
		slog.Error("can't generate token", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"token": token,
	})
}

func (h *Handler) TryIdentify() echo.MiddlewareFunc {
	return h.UserIdentity(true)
}

func (h *Handler) MustIdentify() echo.MiddlewareFunc {
	return h.UserIdentity(false)
}

func (h *Handler) UserIdentity(skiperr bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log := slog.With(slog.String("op", "handler.UserIdentify"), slog.String("request_id", requestid.Get(c)))

			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			if authHeader == "" {
				log.Debug("auth header is empty, skipping check")
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Debug("invalid auth header format, skipping check")
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			tokenString := parts[1]

			token, err := jwtgo.ParseWithClaims(tokenString, &jwt.CustomClaims{}, func(token *jwtgo.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwtgo.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
				}
				return []byte(h.config.TonProof.PayloadSignatureKey), nil
			})
			if err != nil || !token.Valid {
				log.Debug("invalid or expired token", sl.Err(err))
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			claims := token.Claims.(*jwt.CustomClaims)

			c.Set("account", *claims)

			return next(c)
		}
	}
}
