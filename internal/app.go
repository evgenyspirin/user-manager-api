package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"user-manager-api/config"
	"user-manager-api/internal/application/ports"
	"user-manager-api/internal/application/services"
	"user-manager-api/internal/infrastructure/db/postgres"
	"user-manager-api/internal/infrastructure/db/postgres/user"
	"user-manager-api/internal/infrastructure/db/postgres/user_file"
	"user-manager-api/internal/infrastructure/jwt"
	"user-manager-api/internal/infrastructure/metrics"
	"user-manager-api/internal/infrastructure/mq"
	"user-manager-api/internal/infrastructure/s3"
	"user-manager-api/internal/interface/api/rest"
	"user-manager-api/internal/interface/api/rest/middleware"
	"user-manager-api/pkg/rmqconsumer"
)

type App struct {
	logger     *zap.Logger
	cfg        config.Config
	db         *pgxpool.Pool
	s3         ports.S3Client
	httpSrv    *http.Server
	router     *gin.Engine
	mCounter   *prometheus.CounterVec
	mq         ports.RabbitMQ
	mqConsumer ports.RMQConsumer
}

func NewApp(ctx context.Context) (*App, error) {
	// logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// config
	if err = godotenv.Load(".env"); err != nil {
		logger.Fatal("error loading .env file", zap.Error(err))
	}
	cfg := config.Load()

	// metrics
	mCounter := metrics.NewCounter()

	// router
	switch cfg.App.Env {
	case gin.ReleaseMode, "prod", "production":
		gin.SetMode(gin.ReleaseMode)
	case gin.TestMode:
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogGin(logger, mCounter))

	// httpServer
	httpSrv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	// db
	dbDsn, err := cfg.DBDSN()
	if err != nil {
		logger.Fatal("DB config error", zap.Error(err))
	}
	dbPool, err := postgres.New(ctx, logger, dbDsn)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	// s3
	s3Client, err := s3.New(ctx, logger, cfg.S3)
	if err != nil {
		logger.Fatal("failed to connect to S3", zap.Error(err))
	}

	// rabbitMQ
	rabbitDsn, err := cfg.AMQPDSN()
	if err != nil {
		logger.Fatal("RabbitMQ config error", zap.Error(err))
	}
	rbMQ := mq.New(cfg.MQ, logger)
	if err = rbMQ.Connect(ctx, rabbitDsn); err != nil {
		logger.Fatal("failed to connect to rabbitMQ", zap.Error(err))
	}
	if err = rbMQ.Init(); err != nil {
		logger.Fatal("failed init rabbitMQ", zap.Error(err))
	}
	//rmqConsumer
	rmqConsumer := rmqconsumer.New(cfg.MQ, logger, rbMQ.GetConn())
	if err = rmqConsumer.Connect(rabbitDsn); err != nil {
		logger.Fatal("failed to connect rabbitMQ consumer", zap.Error(err))
	}
	if err = rmqConsumer.Init(); err != nil {
		logger.Fatal("failed to init rabbitMQ consumer", zap.Error(err))
	}

	return &App{
		logger:     logger,
		cfg:        cfg,
		db:         dbPool,
		s3:         s3Client,
		httpSrv:    httpSrv,
		router:     r,
		mCounter:   mCounter,
		mq:         rbMQ,
		mqConsumer: rmqConsumer,
	}, nil
}

func (a *App) Close() {
	if a.db != nil {
		a.db.Close()
	}
	if a.mq.GetConn() != nil {
		a.mq.GetConn().Close()
	}
	if a.logger != nil {
		_ = a.logger.Sync()
	}
}

// Run - The central place to launch and manage our application and
// parallel processes through a single context.
func (a *App) Run(ctx context.Context) error {
	// context with os signals cancel chan
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
	defer stop()

	// "errgroup" instead of "WaitGroup" because:
	// - allows return an error from gorutine
	// - group errors from multiple gorutines into one
	// - wg.Add(1), wg.Done() - automatically under the hood, so never catch deadlock if you forget something ;-)
	// - allows orchestration of parallel processes through the context.Context(gracefull shut down)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		a.logger.Info("starting "+a.cfg.App.Name, zap.String("addr", a.cfg.App.Host+":"+a.cfg.App.Port))
		if err := a.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server "+a.cfg.App.Name+" error: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		a.mq.PublisherWorker(ctx)
		return nil
	})

	g.Go(func() error {
		a.mqConsumer.DeliveryWorker(ctx)
		return nil
	})

	<-ctx.Done()

	a.logger.Info("shutting down " + a.cfg.App.Name + " gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if a.httpSrv != nil {
		if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("http server shutdown "+a.cfg.App.Name+" error", zap.Error(err))
			return err
		}
	}

	if err := g.Wait(); err != nil {
		a.logger.Error(a.cfg.App.Name+" returning an error", zap.Error(err))
		return err
	}

	a.logger.Info(a.cfg.App.Name + " gracefully stopped")

	return nil
}

func (a *App) InitControllers() {
	// repos
	userRepo := user.NewRepository(a.db)
	userFileRepo := user_file.NewRepository(a.db)

	// services
	jwtService := jwt.New(a.cfg.App.JWTSecret)
	authService := services.NewAuthService(jwtService)
	userService := services.NewUserService(userRepo, userFileRepo, a.mq, a.mCounter)
	userFileService := services.NewUserFileService(a.s3, userFileRepo, userRepo, a.mCounter)

	// controllers
	rest.NewAuthController(a.router, a.logger, userService, authService)
	rest.NewUserController(a.router, userService, a.logger, jwtService)
	rest.NewUserFileController(a.router, userFileService, a.logger, jwtService)

	// ops
	a.router.GET(rest.RouteHealth, func(c *gin.Context) { c.Status(http.StatusOK) })
	a.router.GET(rest.RouteMetrics, gin.WrapH(promhttp.Handler()))
}

func (a *App) Logger() *zap.Logger { return a.logger }
