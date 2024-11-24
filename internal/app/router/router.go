package router

import (
	"net/http"

	"github.com/cascadecontests/backend/internal/app/handler"
	"github.com/cascadecontests/backend/internal/config"
	"github.com/cascadecontests/backend/pkg/requestid"
	"github.com/labstack/echo/v4"
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

	router.Use(requestid.New)

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
		}
	}

	return router
}
