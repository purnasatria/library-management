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
	pb_author "github.com/purnasatria/library-management/api/gen/author"
	pb_book "github.com/purnasatria/library-management/api/gen/book"
	pb_category "github.com/purnasatria/library-management/api/gen/category"
	"github.com/purnasatria/library-management/internal/book"
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
	GRPCPort               string
	RESTPort               string
	AuthServiceAddress     string
	AuthorServiceAddress   string
	CategoryServiceAddress string
}

func main() {
	// INFO: Set up logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// INFO: Set up flags
	flag.Parse()

	// INFO: setup env
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("Error loading .env file")
	}

	// INFO: Set up database connection
	dbConfig := database.Config{
		URL:             env.Get("BOOK_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/library_book?sslmode=disable"),
		MaxOpenConns:    env.GetInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    env.GetInt("DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: env.GetDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}

	db, err := database.NewConnection(&dbConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// INFO: Migrations
	if *migrate {
		if err := database.RunMigrations(db, "migrations/book"); err != nil {
			log.Fatal().Err(err).Msg("Failed to run migrations")
		}
		log.Info().Msg("Migrations completed successfully")
		return
	}

	// INFO: Set up services
	serverConfig := &ServerConfig{
		GRPCPort:               env.Get("BOOK_GRPC_PORT", ":50054"),
		RESTPort:               env.Get("BOOKREST_PORT", ":8084"),
		AuthServiceAddress:     env.Get("AUTH_SERVICE_ADDRESS", "localhost:50051"),
		AuthorServiceAddress:   env.Get("AUTHOR_SERVICE_ADDRESS", "localhost:50052"),
		CategoryServiceAddress: env.Get("CATEGORY_SERVICE_ADDRESS", "localhost:50053"),
	}

	serverKey := env.Get("SERVER_KEY", "default-server-key")

	// INFO: Create connections to other services
	authConn, err := grpc.NewClient(
		serverConfig.AuthServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcprotocol.ClientServerKeyInterceptor(serverKey)),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to auth service")
	}
	defer authConn.Close()

	authorConn, err := grpc.NewClient(
		serverConfig.AuthorServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcprotocol.ClientServerKeyInterceptor(serverKey)),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to author service")
	}
	defer authorConn.Close()

	categoryConn, err := grpc.NewClient(
		serverConfig.CategoryServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcprotocol.ClientServerKeyInterceptor(serverKey)),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to category service")
	}
	defer categoryConn.Close()

	// INFO: Create service clients
	authClient := pb_auth.NewAuthServiceClient(authConn)
	authorClient := pb_author.NewAuthorServiceClient(authorConn)
	categoryClient := pb_category.NewCategoryServiceClient(categoryConn)

	// INFO: Create book repository and service
	repo := book.NewRepository(db)
	service := book.NewService(repo, authorClient, categoryClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *grpcOnly || (!*httpOnly) {
		go server.RunGRPCServer(server.GRPCServerConfig{
			Port: serverConfig.GRPCPort,
			RegisterService: func(s *grpc.Server) {
				pb_book.RegisterBookServiceServer(s, service)
			},
			UnaryInterceptors: []grpc.UnaryServerInterceptor{
				grpcprotocol.LogInterceptor,
				grpcprotocol.ServerKeyInterceptor(serverKey),
			},
		})
	}

	if *httpOnly || (!*grpcOnly) {
		// Define exempt paths
		exemptPaths := []string{
			"/docs",         // Swagger UI
			"/swagger.json", // Swagger JSON
			"/swagger-ui/",  // Swagger UI assets
		}
		// Create JWT middleware
		jwtMiddleware := httpprotocol.JWTAuthMiddleware(authClient, exemptPaths)

		go server.RunHTTPServer(ctx, server.HTTPServerConfig{
			Port:     serverConfig.RESTPort,
			GRPCPort: serverConfig.GRPCPort,
			Middlewares: []func(http.Handler) http.Handler{
				jwtMiddleware,
			},
			RegisterGateway: func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
				opts = append(opts,
					grpc.WithUnaryInterceptor(grpcprotocol.ClientServerKeyInterceptor(serverKey)),
				)
				return pb_book.RegisterBookServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
			},
			SwaggerUIDir:    "./node_modules/swagger-ui-dist",
			SwaggerJSONPath: "./api/swagger/book.swagger.json",
		})
	}

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	log.Info().Msg("Server exited")
}
