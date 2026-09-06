package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"github.com/oziev02/help-desk/internal/config"
	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/handler"
	"github.com/oziev02/help-desk/internal/middleware"
	"github.com/oziev02/help-desk/internal/repository"
	"github.com/oziev02/help-desk/internal/service"
)

type App struct {
	cfg    config.Config
	db     *repository.Postgres
	server *http.Server
	logger *slog.Logger
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := repository.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if cfg.SeedDemo {
		if err := db.EnsureDemoUsers(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("seed demo users: %w", err)
		}
	}

	authSvc := service.NewAuthService(db, cfg)
	notifier := service.NewLogNotifier(logger)
	ticketSvc := service.NewTicketService(db, notifier, logger, cfg.SLAHours)
	adminSvc := service.NewAdminService(db)

	authHandler := handler.NewAuthHandler(authSvc, logger)
	ticketHandler := handler.NewTicketHandler(ticketSvc, logger)
	adminHandler := handler.NewAdminHandler(adminSvc, logger)

	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(slogchi.NewWithConfig(logger, slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
		Filters: []slogchi.Filter{
			slogchi.IgnorePath("/health"),
		},
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/register", authHandler.Register)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret, db))

			r.Get("/categories", adminHandler.ListCategories)
			r.Get("/rooms", adminHandler.ListRooms)

			r.Route("/tickets", func(r chi.Router) {
				r.Get("/", ticketHandler.List)
				r.Post("/", ticketHandler.Create)
				r.Get("/{id}", ticketHandler.Get)
				r.Patch("/{id}", ticketHandler.Update)
				r.Delete("/{id}", ticketHandler.Delete)
				r.Post("/{id}/comments", ticketHandler.AddComment)
				r.Post("/{id}/assign", ticketHandler.Assign)
				r.Post("/{id}/transitions", ticketHandler.Transition)
				r.Post("/{id}/complete", ticketHandler.Complete)
				r.Post("/{id}/refuse", ticketHandler.Refuse)
				r.Post("/{id}/reopen", ticketHandler.Reopen)
				r.Post("/{id}/cancel", ticketHandler.Cancel)
				r.Post("/{id}/links", ticketHandler.Link)
				r.Delete("/{id}/links/{linkedId}", ticketHandler.Unlink)
			})

			r.Get("/reports/tickets", ticketHandler.Report)

			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin))
				r.Get("/users/{id}/roles", adminHandler.ListUserRoles)
				r.Post("/users/{id}/roles", adminHandler.GrantRole)
				r.Delete("/users/{id}/roles/{role}", adminHandler.RevokeRole)
				r.Get("/categories", adminHandler.ListCategories)
				r.Post("/categories", adminHandler.CreateCategory)
				r.Patch("/categories/{id}", adminHandler.DeactivateCategory)
				r.Get("/rooms", adminHandler.ListRooms)
				r.Post("/rooms", adminHandler.CreateRoom)
				r.Patch("/rooms/{id}", adminHandler.UpdateRoom)
				r.Delete("/rooms/{id}", adminHandler.DeactivateRoom)
			})
		})
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{cfg: cfg, db: db, server: srv, logger: logger}, nil
}

func (a *App) Handler() http.Handler {
	return a.server.Handler
}

func (a *App) Close() {
	a.db.Close()
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("server starting", "addr", a.cfg.HTTPAddr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	a.db.Close()
	a.logger.Info("server stopped")
	return nil
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
