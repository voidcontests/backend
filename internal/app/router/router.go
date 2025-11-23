package router

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/voidcontests/api/internal/app/handler"
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/lib/crypto"
	"github.com/voidcontests/api/internal/storage/broker"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ratelimit"
	"github.com/voidcontests/api/pkg/requestid"
	"github.com/voidcontests/api/pkg/requestlog"
	"github.com/voidcontests/api/pkg/ton"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(c *config.Config, r *repository.Repository, b broker.Broker, tc *ton.Client, cipher crypto.Cipher) *Router {
	h := handler.New(c, r, b, tc, cipher)
	return &Router{config: c, handler: h}
}

func (r *Router) InitRoutes() *echo.Echo {
	router := echo.New()

	router.HTTPErrorHandler = handler.ErorHTTP

	router.Use(requestid.New)
	router.Use(requestlog.Completed)
	router.Pre(middleware.RemoveTrailingSlash())

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

	// TODO: update rate limiting logic:
	//   Current:
	//     - request -> wait Ns -> request
	//
	//   Expected:
	//     - [request -> request -> request] - in such window, forbid to make more than M requests
	//       ^ 0s                       Ns ^

	api := router.Group("/api")
	{
		api.GET("/healthcheck", r.handler.Healthcheck)

		tonproof := api.Group("/tonproof")
		tonproof.POST("/payload", r.handler.GeneratePayload)
		tonproof.POST("/check", r.handler.CheckProof, r.handler.MustIdentify())

		api.GET("/account", r.handler.GetAccount, r.handler.MustIdentify())
		api.POST("/account", r.handler.CreateAccount, ratelimit.WithTimeout(5*time.Second))
		api.PATCH("/account", r.handler.UpdateAccount, r.handler.MustIdentify())
		api.POST("/session", r.handler.CreateSession, ratelimit.WithTimeout(2*time.Second))

		api.GET("/account/contests", r.handler.GetCreatedContests, r.handler.MustIdentify())
		api.GET("/account/problems", r.handler.GetCreatedProblems, r.handler.MustIdentify())

		api.POST("/problems", r.handler.CreateProblem, ratelimit.WithTimeout(3*time.Second), r.handler.MustIdentify())

		api.GET("/problems/:pid", r.handler.GetProblemByID, r.handler.MustIdentify())

		api.GET("/contests", r.handler.GetContests)
		api.POST("/contests", r.handler.CreateContest, ratelimit.WithTimeout(3*time.Second), r.handler.MustIdentify())

		api.GET("/contests/:cid", r.handler.GetContestByID, r.handler.TryIdentify())
		api.POST("/contests/:cid/entry", r.handler.CreateEntry, ratelimit.WithTimeout(3*time.Second), r.handler.MustIdentify())
		api.GET("/contests/:cid/scores", r.handler.GetScores)

		api.GET("/contests/:cid/problems/:charcode", r.handler.GetContestProblem, r.handler.MustIdentify())
		api.GET("/contests/:cid/problems/:charcode/submissions", r.handler.GetSubmissions, r.handler.MustIdentify())
		api.POST("/contests/:cid/problems/:charcode/submissions",
			r.handler.CreateSubmission, ratelimit.WithTimeout(2*time.Second), r.handler.MustIdentify())
		api.GET("/submissions/:sid", r.handler.GetSubmissionByID, r.handler.MustIdentify())
	}

	return router
}
