package main

import (
	"context"
	"google.golang.org/grpc"
	"log/slog"
	"mod1/config"
	authserver "mod1/internal/server/auth"
	taskserver "mod1/internal/server/task"
	authserv "mod1/internal/services/auth"
	taskserv "mod1/internal/services/task"
	"mod1/internal/storage"
	authandtaskv1 "mod1/proto/gen/go"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envProd = "prod"
	envDev  = "dev" // исправлено на lowercase
)

var (
	TokenTTL        = 1 * time.Hour
	ShutdownTimeout = 10 * time.Second // можно вынести в конфиг
)

func main() {
	log := SetupLogger(envDev)
	log.Info(
		"starting server",
		slog.String("env", envDev),
		slog.String("version", "0.0.1"),
	)

	cfg := config.MustLoad()

	// Инициализация хранилища
	db, err := storage.New(cfg.DBConf)
	if err != nil {
		log.Error("failed to init storage",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Starage init")
	// Инициализация сервисов
	authService := authserv.New(log, db, db, TokenTTL)
	taskService := taskserv.NewTaskService(db)

	// Настройка gRPC сервера
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authserver.AuthInterceptor),
	)
	authandtaskv1.RegisterAuthServiceServer(grpcServer, &authserver.AuthServer{AuthService: authService})
	authandtaskv1.RegisterTaskServiceServer(grpcServer, &taskserver.TaskServer{Service: taskService})

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера
	listener, err := net.Listen("tcp", cfg.ServConf.HostgRPC)
	if err != nil {
		log.Error("failed to create listener",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		log.Info("starting gRPC server",
			slog.String("address", cfg.ServConf.HostgRPC))
		if err := grpcServer.Serve(listener); err != nil {
			log.Error("failed to serve gRPC",
				slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Ожидание сигнала завершения
	<-done
	log.Info("server is shutting down...")

	// Graceful stop
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Info("server stopped gracefully")
	case <-ctx.Done():
		log.Warn("forcing server shutdown due to timeout")
		grpcServer.Stop()
	}

	log.Info("server shutdown completed")
}

func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
