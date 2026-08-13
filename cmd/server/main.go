// Package main is the composition root. It wires all dependencies — config,
// database, repositories, services, use cases, and handlers — then starts
// the HTTP server. No business logic lives here.

// @title          智索 API
// @version         1.0
// @description    智索后端服务 API
// @host           localhost:8080
// @BasePath       /api/v1
// @schemes        http
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/kun/zhisuo-server/docs"

	articleHandler "github.com/kun/zhisuo-server/internal/article/adapter/handler"
	articleRepo "github.com/kun/zhisuo-server/internal/article/adapter/repository"
	articleService "github.com/kun/zhisuo-server/internal/article/adapter/service"
	articleEntity "github.com/kun/zhisuo-server/internal/article/entity"
	articleUsecase "github.com/kun/zhisuo-server/internal/article/usecase"

	commentHandler "github.com/kun/zhisuo-server/internal/comment/adapter/handler"
	commentRepo "github.com/kun/zhisuo-server/internal/comment/adapter/repository"
	commentEntity "github.com/kun/zhisuo-server/internal/comment/entity"
	commentUsecase "github.com/kun/zhisuo-server/internal/comment/usecase"

	"github.com/kun/zhisuo-server/internal/infrastructure"
	infraCache "github.com/kun/zhisuo-server/internal/infrastructure/cache"

	userHandler "github.com/kun/zhisuo-server/internal/user/adapter/handler"
	userRepo "github.com/kun/zhisuo-server/internal/user/adapter/repository"
	userSvc "github.com/kun/zhisuo-server/internal/user/adapter/service"
	userEntity "github.com/kun/zhisuo-server/internal/user/entity"
	userUsecase "github.com/kun/zhisuo-server/internal/user/usecase"

	"github.com/kun/zhisuo-server/internal/port"
)

// main loads configuration, initializes infrastructure, wires all layers,
// runs schema migration, and starts the HTTP server with graceful shutdown.
func main() {
	configPath := flag.String("config", "", "path to config YAML file (default: config.yaml or config/development.yaml)")
	flag.Parse()

	cfg := infrastructure.LoadConfig(*configPath)
	infrastructure.InitLogger(cfg)
	infrastructure.InitMetrics()

	db, err := infrastructure.NewDB(cfg)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	}()

	if err := infrastructure.Migrate(db, &userEntity.User{}, &articleEntity.Article{}, &commentEntity.Comment{}); err != nil {
		slog.Error("schema migration failed", "error", err)
		os.Exit(1)
	}

	uRepo := userRepo.NewUserMySQL(db)
	aRepo := articleRepo.NewArticleMySQL(db)
	cRepo := commentRepo.NewCommentMySQL(db)

	// Cache-aside decorator is applied only when caching is enabled in config.
	var aRepoEffective articleUsecase.Repository = aRepo
	if cfg.CacheEnabled {
		cache := infraCache.NewCache(cfg)
		aRepoEffective = articleRepo.NewArticleCache(aRepo, cache, cfg.CacheTTL)
	}

	txManager := infrastructure.NewTxManager(db)

	userService := userSvc.NewUserService(uRepo)
	articleService := articleService.NewArticleService(aRepoEffective)

	userUseCase := userUsecase.NewUserUseCase(uRepo)
	articleUseCase := articleUsecase.NewArticleUseCase(aRepoEffective, userService)
	commentUseCase := commentUsecase.NewCommentUseCase(cRepo, txManager, articleService, userService)

	pageCfg := port.PageConfig{DefaultSize: cfg.DefaultPageSize, MaxSize: cfg.MaxPageSize}

	userAPI := userHandler.NewUserHandler(userUseCase, pageCfg)
	articleAPI := articleHandler.NewArticleHandler(articleUseCase, pageCfg)
	commentAPI := commentHandler.NewCommentHandler(commentUseCase, pageCfg)

	router := infrastructure.NewRouter(infrastructure.Handlers{
		User:    userAPI,
		Article: articleAPI,
		Comment: commentAPI,
	}, db, infrastructure.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst))

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: cfg.ServerReadHdrT,
		ReadTimeout:       cfg.ServerReadT,
		WriteTimeout:      cfg.ServerWriteT,
		IdleTimeout:       cfg.ServerIdleT,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}
