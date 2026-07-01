package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/rest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"

	_ "github.com/mattn/go-sqlite3"
)

// songUpdaterAdapter 桥接 REST handler 的 SongUpdater 到 SongService。
type songUpdaterAdapter struct {
	svc *song.Service
}

func (a *songUpdaterAdapter) UpdateFromScan(songID string, meta *metadata.AudioMetadata, fileSize int64) error {
	err := a.svc.UpdateFromScan(context.Background(), songID,
		meta.Title, meta.Artist, meta.Album, meta.Genre,
		int32(meta.TrackNumber), int32(meta.DiscNumber), int32(meta.Year),
		meta.FileHash, meta.FileName, meta.MIMEType, fileSize)
	return err
}

func main() {
	viper.SetDefault("grpc_port", 9090)
	viper.SetDefault("rest_port", 9091)
	viper.SetDefault("db_path", "data/echovault.db")
	viper.SetDefault("jwt_secret", "change-me-in-production")
	viper.SetDefault("storage_type", "local")
	viper.SetDefault("storage_path", "data/files")
	viper.AutomaticEnv()

	dbPath := viper.GetString("db_path")
	drv, err := entsql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_fk=1")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer drv.Close()

	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	slog.Info("database migrated successfully")

	jwtSecret := viper.GetString("jwt_secret")

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("grpc_port")))
	if err != nil {
		log.Fatalf("failed to listen gRPC: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(evgrpc.AuthInterceptor(jwtSecret)))
	evgrpc.RegisterAll(s, client, jwtSecret)

	// 启动 REST 文件服务 (Gin)
	storageSvc, err := storage.NewStorage(
		viper.GetString("storage_type"),
		viper.GetString("storage_path"),
	)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	// 创建 SongUpdater adapter
	songSvc := song.NewService(client)
	songUpdater := &songUpdaterAdapter{svc: songSvc}

	restHandler := rest.NewHandler(storageSvc, songUpdater)
	ginServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("rest_port")),
		Handler: restHandler,
	}

	// 优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down servers...")
		cancel()
		s.GracefulStop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		ginServer.Shutdown(shutdownCtx)
	}()

	go func() {
		slog.Info("starting REST file server", "port", viper.GetInt("rest_port"))
		if err := ginServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST server error: %v", err)
		}
	}()

	slog.Info("starting EchoVault server", "port", viper.GetInt("grpc_port"))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
