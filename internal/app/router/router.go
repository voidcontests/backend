package router

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/backend/internal/app/handler"
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/requestlog"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(c *config.Config, r *repository.Repository, mainnet, testnet *tonconnect.Server) *Router {
	h := handler.New(c, r, mainnet, testnet)
	return &Router{config: c, handler: h}
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
	router.Pre(middleware.RemoveTrailingSlash())

	// TODO: use custom validator with e.Validator

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

	jwtopts := middleware.JWTConfig{
		Claims:     &jwt.CustomClaims{},
		SigningKey: []byte(r.config.TonProof.PayloadSignatureKey),
		ContextKey: "account",
	}

	api := router.Group("/api")
	{
		api.GET("/healthcheck", r.handler.Healthcheck)

		tonproof := api.Group("/tonproof")
		{
			tonproof.POST("/payload", r.handler.GeneratePayload)
			tonproof.POST("/check", r.handler.CheckProof)

			// TODO: Migrate to `echo-jwt` middleware
			tonproof.GET("/account", r.handler.GetAccount, middleware.JWTWithConfig(jwtopts))
		}

		contests := api.Group("/contests")
		{
			contests.GET("", r.handler.GetContests)
			contests.POST("", r.handler.CreateContest, middleware.JWTWithConfig(jwtopts))
			contests.GET("/:id", r.handler.GetContestByID)
		}

		problems := api.Group("/problems")
		{
			problems.GET("", r.handler.GetProblems)
		}
	}

	return router
}
