package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	pb "github.com/purnasatria/library-management/api/gen/auth"
	"github.com/purnasatria/library-management/internal/auth"
	"github.com/purnasatria/library-management/pkg/database"
	"github.com/purnasatria/library-management/pkg/jwt"
	grpcprotocol "github.com/purnasatria/library-management/pkg/protocol/grpc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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
		URL:             getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/library?sslmode=disable"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
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
		AccessTokenSecret:          getEnv("JWT_ACCESS_SECRET", ""),
		AccessTokenExpirationTime:  getEnvAsDuration("JWT_ACCESS_EXPIRATION_TIME", 15*time.Minute),
		RefreshTokenSecret:         getEnv("JWT_REFRESH_SECRET", ""),
		RefreshTokenExpirationTime: getEnvAsDuration("JWT_REFRESH_EXPIRATION_TIME", 7*24*time.Hour),
	}

	// INFO: setup service
	servercfg := &ServerConfig{
		GRPCPort: getEnv("GRPC_PORT", ":50051"),
		RESTPort: getEnv("REST_PORT", ":8081"),
	}
	jwtManager := jwt.New(jwtcfg)
	repo := auth.NewRepository(db)
	service := auth.NewService(repo, jwtManager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *grpcOnly || (!*httpOnly) {
		go runGRPCServer(servercfg, service)
	}

	if *httpOnly || (!*grpcOnly) {
		go runHTTPServer(ctx, servercfg)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Warn().Msg("Shutting down server...")
}

func runGRPCServer(cfg *ServerConfig, service *auth.Service) {
	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to listen on port: %v", cfg.GRPCPort)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(grpcprotocol.LogInterceptor))
	pb.RegisterAuthServiceServer(s, service)

	log.Info().Msgf("Starting gRPC server on %s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("Failed to serve gRPC")
	}
}

func runHTTPServer(ctx context.Context, cfg *ServerConfig) {
	mux := http.NewServeMux()

	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(runtime.DefaultHTTPErrorHandler),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			return metadata.Pairs("grpc-status-details-bin", req.Header.Get("Grpc-Metadata-Grpc-Status-Details-Bin"))
		}),
	)
	optsgrpc := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterAuthServiceHandlerFromEndpoint(ctx, gwmux, cfg.GRPCPort, optsgrpc)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to register gateway")
	}

	// Mount the gRPC-Gateway at /api/v1
	// mux.Handle("/api/", http.StripPrefix("/api", gwmux))
	// mux.Handle("/api/v1", gwmux)
	mux.Handle("/", gwmux)

	// Setup Swagger UI
	swaggerUiDir := "./node_modules/swagger-ui-dist"
	swaggerJsonPath := "./api/swagger/auth.swagger.json"

	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerJsonPath)
	})

	opts := middleware.SwaggerUIOpts{
		SpecURL: "/swagger.json",
		Path:    "docs",
	}
	sh := middleware.SwaggerUI(opts, nil)
	mux.Handle("/docs", sh)

	fs := http.FileServer(http.Dir(swaggerUiDir))
	mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui", fs))

	log.Info().Msgf("Starting HTTP server on %s", cfg.RESTPort)
	log.Info().Msgf("API is served at http://localhost%s/api", cfg.RESTPort)
	log.Info().Msgf("Swagger UI is served at http://localhost%s/docs", cfg.RESTPort)

	if err := http.ListenAndServe(cfg.RESTPort, mux); err != nil {
		log.Fatal().Err(err).Msg("Failed to serve HTTP")
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
