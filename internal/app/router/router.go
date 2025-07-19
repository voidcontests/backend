package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/voidcontests/backend/internal/app/handler"
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository"
	"github.com/voidcontests/backend/pkg/ratelimit"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/requestlog"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(c *config.Config, r *repository.Repository) *Router {
	h := handler.New(c, r)
	return &Router{config: c, handler: h}
}

func (r *Router) InitRoutes() *echo.Echo {
	router := echo.New()

	router.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok && (he.Code == http.StatusNotFound || he.Code == http.StatusMethodNotAllowed) {
			c.JSON(http.StatusNotFound, map[string]string{
				"message": "resource not found",
			})
			return
		}

		if ae, ok := err.(*handler.APIError); ok {
			slog.Debug("responded with API error", sl.Err(err), slog.String("request_id", requestid.Get(c)))
			c.JSON(ae.Status, map[string]any{
				"message": ae.Message,
			})
			return
		}

		slog.Error("something went wrong", sl.Err(err), slog.String("request_id", requestid.Get(c)))
		c.JSON(http.StatusInternalServerError, map[string]any{
			"message": "internal server error",
		})
	}

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

	api := router.Group("/api")
	{
		api.GET("/healthcheck", r.handler.Healthcheck)

		api.GET("/account", r.handler.GetAccount, r.handler.MustIdentify())
		api.POST("/account", r.handler.CreateAccount)
		api.POST("/session", r.handler.CreateSession)

		// TODO: make this endpoints as filter to general endpoint, like:
		// GET /contests?creator_id=69
		// GET /problems?writer_id=420
		api.GET("/creator/contests", r.handler.GetCreatedContests, r.handler.MustIdentify())
		api.GET("/creator/problems", r.handler.GetCreatedProblems, r.handler.MustIdentify())

		api.POST("/problems", r.handler.CreateProblem, r.handler.MustIdentify())

		api.GET("/problems/:pid", r.handler.GetProblemByID, r.handler.MustIdentify())

		api.GET("/contests", r.handler.GetContests)
		api.POST("/contests", r.handler.CreateContest, r.handler.MustIdentify())

		api.GET("/contests/:cid", r.handler.GetContestByID, r.handler.TryIdentify())
		api.POST("/contests/:cid/entry", r.handler.CreateEntry, r.handler.MustIdentify())
		api.GET("/contests/:cid/leaderboard", r.handler.GetLeaderboard)

		api.GET("/contests/:cid/problems/:charcode", r.handler.GetContestProblem, r.handler.MustIdentify())
		api.GET("/contests/:cid/problems/:charcode/submissions", r.handler.GetSubmissions, r.handler.MustIdentify())
		api.POST("/contests/:cid/problems/:charcode/submissions",
			r.handler.CreateSubmission, ratelimit.WithTimeout(5*time.Second), r.handler.MustIdentify())
		api.GET("/submissions/:sid", r.handler.GetSubmissionByID, r.handler.MustIdentify())
	}

	return router
}
