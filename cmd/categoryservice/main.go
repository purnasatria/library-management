package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	pb_auth "github.com/purnasatria/library-management/api/gen/auth"
	pb_category "github.com/purnasatria/library-management/api/gen/category"
	"github.com/purnasatria/library-management/internal/category"
	"github.com/purnasatria/library-management/pkg/database"
	"github.com/purnasatria/library-management/pkg/env"
	grpcprotocol "github.com/purnasatria/library-management/pkg/protocol/grpc"
	httpprotocol "github.com/purnasatria/library-management/pkg/protocol/http"
	"github.com/purnasatria/library-management/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcOnly = flag.Bool("grpc", false, "Run gRPC server only")
	httpOnly = flag.Bool("http", false, "Run HTTP server only")
	migrate  = flag.Bool("migrate", false, "Run database migrations")
)

type ServerConfig struct {
	GRPCPort           string
	RESTPort           string
	AuthServiceAddress string
}

func main() {
	// INFO: Set up logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// INFO: setup flags
	flag.Parse()

	// INFO: setup env
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("Error loading .env file")
	}

	// INFO: setup db
	dbConfig := database.Config{
		URL:             env.Get("CATEGORY_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/library_category?sslmode=disable"),
		MaxOpenConns:    env.GetInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    env.GetInt("DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: env.GetDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}

	db, err := database.NewConnection(&dbConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// INFO: Migration
	if *migrate {
		if err := database.RunMigrations(db, "migrations/category"); err != nil {
			log.Fatal().Err(err).Msg("Failed to run migrations")
		}
		log.Info().Msg("Migrations completed successfully")
		return
	}

	// INFO: setup service
	servercfg := &ServerConfig{
		GRPCPort:           env.Get("CATEGORY_GRPC_PORT", ":50053"),
		RESTPort:           env.Get("CATEGORY_REST_PORT", ":8083"),
		AuthServiceAddress: env.Get("AUTH_SERVICE_ADDRESS", "localhost:50051"),
	}

	serverKey := env.Get("SERVER_KEY", "default-server-key")

	// INFO: Create a connection to the auth service
	authConn, err := grpc.NewClient(
		servercfg.AuthServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcprotocol.ClientServerKeyInterceptor(serverKey)),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to auth service")
	}
	defer authConn.Close()
	state := authConn.GetState()
	log.Info().Str("state", state.String()).Msg("Connected to auth service")

	// INFO: Create the auth service client
	authClient := pb_auth.NewAuthServiceClient(authConn)

	// Create category repository and service
	repo := category.NewRepository(db)
	service := category.NewService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *grpcOnly || (!*httpOnly) {
		go server.RunGRPCServer(server.GRPCServerConfig{
			Port: servercfg.GRPCPort,
			RegisterService: func(s *grpc.Server) {
				pb_category.RegisterCategoryServiceServer(s, service)
			},
			UnaryInterceptors: []grpc.UnaryServerInterceptor{
				grpcprotocol.LogInterceptor,
				grpcprotocol.ServerKeyInterceptor(serverKey),
			},
		})
	}

	if *httpOnly || (!*grpcOnly) {
		// INFO: Define exempt paths
		exemptPaths := []string{
			"/docs",         // Swagger UI
			"/swagger.json", // Swagger JSON
			"/swagger-ui/",  // Swagger UI assets
		}
		// INFO: Create JWT middleware
		jwtMiddleware := httpprotocol.JWTAuthMiddleware(authClient, exemptPaths)

		go server.RunHTTPServer(ctx, server.HTTPServerConfig{
			Port:     servercfg.RESTPort,
			GRPCPort: servercfg.GRPCPort,
			Middlewares: []func(http.Handler) http.Handler{
				jwtMiddleware,
			},
			// RegisterGateway: pb_author.RegisterAuthorServiceHandlerFromEndpoint,
			RegisterGateway: func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
				opts = append(opts,
					grpc.WithUnaryInterceptor(grpcprotocol.ClientServerKeyInterceptor(serverKey)),
				)
				if err := pb_category.RegisterCategoryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
					return err
				}
				return nil
			},
			SwaggerUIDir:    "./node_modules/swagger-ui-dist",
			SwaggerJSONPath: "./api/swagger/category.swagger.json",
		})
	}

	// INFO: simple gracefull shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	log.Info().Msg("Server exited")
}
