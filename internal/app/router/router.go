package router

import (
	"log/slog"
	"net/http"

	"github.com/cascadecontests/backend/internal/app/handler"
	"github.com/cascadecontests/backend/internal/config"
	"github.com/cascadecontests/backend/internal/jwt"
	"github.com/cascadecontests/backend/internal/lib/logger/sl"
	"github.com/cascadecontests/backend/pkg/requestid"
	"github.com/cascadecontests/backend/pkg/requestlog"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tonkeeper/tongo/tonconnect"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(config *config.Config, mainnet, testnet *tonconnect.Server) *Router {
	h := handler.New(config, mainnet, testnet)
	return &Router{config: config, handler: h}
}

func (r *Router) InitRoutes() *echo.Echo {
	router := echo.New()

	router.HTTPErrorHandler = func(err error, c echo.Context) {
		slog.Error("error occurred", sl.Err(err))
		if apiErr, ok := err.(*handler.APIError); ok {
			c.JSON(apiErr.Status, echo.Map{
				"message": apiErr.Message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}

	router.Use(requestid.New)
	router.Use(requestlog.Completed)

	switch r.config.Env {
	case config.EnvLocal, config.EnvDevelopment:
		router.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Access-Control-Allow-Origin", "*")
				c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
				c.Response().Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

				if c.Request().Method == "OPTIONS" {
					return c.NoContent(http.StatusNoContent)
				}

				return next(c)
			}
		})
	}

	api := router.Group("/api")
	{
		api.GET("/healthcheck", r.handler.Healthcheck)

		tonproof := api.Group("/tonproof")
		{
			tonproof.POST("/payload", r.handler.GeneratePayload)
			tonproof.POST("/check", r.handler.CheckProof)

			// TODO: Migrate to `echo-jwt` middleware
			tonproof.GET("/account", r.handler.GetAccount, middleware.JWTWithConfig(middleware.JWTConfig{
				Claims:     &jwt.CustomClaims{},
				SigningKey: []byte(r.config.TonProof.PayloadSignatureKey),
			}))
		}
	}

	return router
}
