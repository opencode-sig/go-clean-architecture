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
	"flag"
	"log/slog"
	"net/http"

	_ "github.com/kun/zhisuo-server/docs"

	articleHandler "github.com/kun/zhisuo-server/internal/article/adapter/handler"
	articleRepo "github.com/kun/zhisuo-server/internal/article/adapter/repository"
	articleService "github.com/kun/zhisuo-server/internal/article/adapter/service"
	articleUsecase "github.com/kun/zhisuo-server/internal/article/usecase"

	commentHandler "github.com/kun/zhisuo-server/internal/comment/adapter/handler"
	commentRepo "github.com/kun/zhisuo-server/internal/comment/adapter/repository"
	commentUsecase "github.com/kun/zhisuo-server/internal/comment/usecase"

	"github.com/kun/zhisuo-server/internal/infrastructure"
	infraCache "github.com/kun/zhisuo-server/internal/infrastructure/cache"

	userHandler "github.com/kun/zhisuo-server/internal/user/adapter/handler"
	userRepo "github.com/kun/zhisuo-server/internal/user/adapter/repository"
	userSvc "github.com/kun/zhisuo-server/internal/user/adapter/service"
	userUsecase "github.com/kun/zhisuo-server/internal/user/usecase"
)

// main loads configuration, initializes infrastructure, wires all layers,
// and starts the HTTP listener. It exits if the database connection fails.
func main() {
	configPath := flag.String("config", "", "path to config YAML file (default: config.yaml or config/development.yaml)")
	flag.Parse()

	cfg := infrastructure.LoadConfig(*configPath)
	infrastructure.InitLogger(cfg)
	infrastructure.InitMetrics()

	db, err := infrastructure.NewDB(cfg.DSN())
	if err != nil {
		slog.Error("database connection failed", "error", err)
		return
	}
	defer func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	}()

	uRepo := userRepo.NewUserMySQL(db)
	aRepo := articleRepo.NewArticleMySQL(db)
	cRepo := commentRepo.NewCommentMySQL(db)

	cache := infraCache.NewCache(cfg)
	aRepoWithCache := articleRepo.NewArticleCache(aRepo, cache, cfg.CacheTTL)

	txManager := infrastructure.NewTxManager(db)

	userService := userSvc.NewUserService(uRepo)
	articleService := articleService.NewArticleService(aRepoWithCache)

	userUseCase := userUsecase.NewUserUseCase(uRepo)
	articleUseCase := articleUsecase.NewArticleUseCase(aRepoWithCache, userService)
	commentUseCase := commentUsecase.NewCommentUseCase(cRepo, txManager, articleService, userService)

	userAPI := userHandler.NewUserHandler(userUseCase)
	articleAPI := articleHandler.NewArticleHandler(articleUseCase)
	commentAPI := commentHandler.NewCommentHandler(commentUseCase)

	router := infrastructure.NewRouter(infrastructure.Handlers{
		User:    userAPI,
		Article: articleAPI,
		Comment: commentAPI,
	})

	slog.Info("server starting", "port", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, router); err != nil {
		slog.Error("server failed", "error", err)
	}
}
