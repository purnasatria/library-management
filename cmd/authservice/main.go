package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	pb "github.com/purnasatria/library-management/api/gen/auth"
	"github.com/purnasatria/library-management/internal/auth"
	"github.com/purnasatria/library-management/pkg/database"
	"github.com/purnasatria/library-management/pkg/env"
	"github.com/purnasatria/library-management/pkg/jwt"
	grpcprotocol "github.com/purnasatria/library-management/pkg/protocol/grpc"
	"github.com/purnasatria/library-management/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var (
	grpcOnly = flag.Bool("grpc", false, "Run gRPC server only")
	httpOnly = flag.Bool("http", false, "Run HTTP server only")
	migrate  = flag.Bool("migrate", false, "Run database migrations")
)

type ServerConfig struct {
	GRPCPort string
	RESTPort string
}

func main() {
	// INFO: Set up the logger
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// INFO: setup flag
	flag.Parse()

	// INFO: setup env
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("can't load config file, load defaults")
	}

	// INFO: setup db
	dbcfg := database.Config{
		URL:             env.Get("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/library?sslmode=disable"),
		MaxOpenConns:    env.GetInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    env.GetInt("DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: env.GetDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}

	db, err := database.NewConnection(&dbcfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// INFO: Migration
	if *migrate {
		if err := database.RunMigrations(db, "migrations/auth"); err != nil {
			log.Fatal().Err(err).Msg("Failed to run migrations")
		}
		log.Info().Msg("Migrations completed successfully")
		return
	}

	// INFO: setup jwt
	jwtcfg := &jwt.Config{
		AccessTokenSecret:          env.Get("JWT_ACCESS_SECRET", ""),
		AccessTokenExpirationTime:  env.GetDuration("JWT_ACCESS_EXPIRATION_TIME", 15*time.Minute),
		RefreshTokenSecret:         env.Get("JWT_REFRESH_SECRET", ""),
		RefreshTokenExpirationTime: env.GetDuration("JWT_REFRESH_EXPIRATION_TIME", 7*24*time.Hour),
	}

	// INFO: setup service
	servercfg := &ServerConfig{
		GRPCPort: env.Get("GRPC_PORT", ":50051"),
		RESTPort: env.Get("REST_PORT", ":8081"),
	}
	jwtManager := jwt.New(jwtcfg)
	repo := auth.NewRepository(db)
	service := auth.NewService(repo, jwtManager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *grpcOnly || (!*httpOnly) {
		go server.RunGRPCServer(server.GRPCServerConfig{
			Port: servercfg.GRPCPort,
			RegisterService: func(s *grpc.Server) {
				pb.RegisterAuthServiceServer(s, service)
			},
			UnaryInterceptors: []grpc.UnaryServerInterceptor{grpcprotocol.LogInterceptor},
		})
	}

	if *httpOnly || (!*grpcOnly) {
		go server.RunHTTPServer(ctx, server.HTTPServerConfig{
			Port:            servercfg.RESTPort,
			GRPCPort:        servercfg.GRPCPort,
			RegisterGateway: pb.RegisterAuthServiceHandlerFromEndpoint,
			SwaggerUIDir:    "./node_modules/swagger-ui-dist",
			SwaggerJSONPath: "./api/swagger/auth.swagger.json",
		})
	}

	// INFO: simple graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Warn().Msg("Shutting down server...")
}
