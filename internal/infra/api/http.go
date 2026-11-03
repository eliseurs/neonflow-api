package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/config"
	"neonflow-api/internal/pkg/middleware"
	"neonflow-api/internal/pkg/security"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

type App struct {
	db      *sql.DB
	pinger  Pinger
	log     *slog.Logger
	router  *gin.Engine
	configs config.Configs
	tokens  *security.TokenProvider
}

// Routes descreve as proteções de um grupo de rotas em /api.
type Routes struct {
	Authorization bool
	Roles         []string
}

func New(configs config.Configs, database *sql.DB, logger *slog.Logger) *App {
	if configs.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	app := &App{
		db:      database,
		log:     logger,
		configs: configs,
		tokens:  security.NewTokenProvider(configs.JWTSecret, time.Duration(configs.JWTExpiryHours)*time.Hour),
	}
	if database != nil {
		app.pinger = database
	}
	return app
}

func (h *App) Setup() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(h.log))
	router.Use(middleware.CORS(h.configs.AllowedOrigins))
	h.router = router
	return router
}

func (h *App) Router() *gin.Engine {
	if h.router == nil {
		return h.Setup()
	}
	return h.router
}

// API devolve um grupo /api com autenticação e papéis conforme Routes.
func (h *App) API(r Routes) *gin.RouterGroup {
	group := h.Router().Group("/api")
	if r.Authorization {
		group.Use(middleware.AuthMiddleware(h.tokens))
	}
	if len(r.Roles) > 0 {
		group.Use(middleware.RoleGuard(r.Roles...))
	}
	return group
}

// HealthCheck responde 200 com o banco acessível e 503 caso contrário (healthcheck do Railway).
func (h *App) HealthCheck() {
	handler := func(c *gin.Context) {
		dbStatus := "ok"
		status := http.StatusOK
		if err := h.pingDB(c.Request.Context()); err != nil {
			dbStatus = "unavailable"
			status = http.StatusServiceUnavailable
		}
		Respond(c, status, gin.H{"status": http.StatusText(status), "database": dbStatus, "time": time.Now().UTC().Format(time.RFC3339)})
	}
	h.Router().GET("/health", handler)
	h.Router().GET("/api/health", handler)
}

func (h *App) pingDB(ctx context.Context) error {
	if h.pinger == nil {
		return errors.New("database not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.pinger.PingContext(ctx)
}

// Server sobe o HTTP server e encerra com graceful shutdown em SIGINT/SIGTERM.
func (h *App) Server() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := h.serve(ctx); err != nil {
		h.log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// serve atende até ctx ser cancelado e então drena as conexões por até 10s.
func (h *App) serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", h.configs.Port),
		Handler:           h.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		h.log.Info("server listening", "port", h.configs.Port, "env", h.configs.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	h.log.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func (h *App) DB() *sql.DB                     { return h.db }
func (h *App) Logger() *slog.Logger            { return h.log }
func (h *App) Configs() config.Configs         { return h.configs }
func (h *App) Tokens() *security.TokenProvider { return h.tokens }

// requestLogger registra método, rota, status e latência. Não loga query string nem corpo (LGPD).
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		log.Info("request", "method", c.Request.Method, "route", route, "status", c.Writer.Status(), "latency_ms", time.Since(start).Milliseconds())
	}
}
