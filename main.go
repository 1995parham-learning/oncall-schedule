package main

import (
	"context"
	"fmt"

	"github.com/1995parham-learning/oncall-schedule/internal/config"
	"github.com/1995parham-learning/oncall-schedule/internal/handler"
	"github.com/1995parham-learning/oncall-schedule/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		fx.Provide(
			// Provide configuration
			config.Load,
			// Provide logger
			zap.NewProduction,
			// Provide storage
			func() storage.Storage {
				return storage.NewMemoryStorage()
			},
			// Provide handler
			handler.New,
			// Provide Echo server
			newEchoServer,
		),
		fx.Invoke(registerRoutes),
		fx.Invoke(startServer),
	)

	app.Run()
}

// newEchoServer creates a new Echo server with middleware.
func newEchoServer(logger *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Add middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				logger.Error("request failed",
					zap.String("uri", v.URI),
					zap.Int("status", v.Status),
					zap.Error(v.Error),
				)
			} else {
				logger.Info("request",
					zap.String("uri", v.URI),
					zap.Int("status", v.Status),
				)
			}
			return nil
		},
	}))

	return e
}

// registerRoutes registers all HTTP routes.
func registerRoutes(e *echo.Echo, h *handler.Handler) {
	e.POST("/schedule", h.CreateSchedule)
	e.GET("/schedule", h.GetSchedule)
}

// startServer starts the HTTP server with graceful shutdown.
func startServer(lc fx.Lifecycle, e *echo.Echo, cfg *config.Config, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
			logger.Info("starting server", zap.String("address", addr))

			// Start server in a goroutine
			go func() {
				if err := e.Start(addr); err != nil {
					logger.Error("server failed", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down server")
			return e.Shutdown(ctx)
		},
	})
}
