package server

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type HTTPServerConfig struct {
	Port            string
	GRPCPort        string
	RegisterGateway func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	SwaggerUIDir    string
	SwaggerJSONPath string
}

func RunHTTPServer(ctx context.Context, cfg HTTPServerConfig) {
	mux := http.NewServeMux()

	// Setup logger
	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(runtime.DefaultHTTPErrorHandler),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			return metadata.Pairs("grpc-status-details-bin", req.Header.Get("Grpc-Metadata-Grpc-Status-Details-Bin"))
		}),
	)
	optsgrpc := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := cfg.RegisterGateway(ctx, gwmux, cfg.GRPCPort, optsgrpc)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to register gateway")
	}

	// Register gateway
	mux.Handle("/", gwmux)

	// Setup Swagger UI
	setupSwaggerUI(mux, cfg.SwaggerUIDir, cfg.SwaggerJSONPath)

	log.Info().Msgf("Starting HTTP server on %s", cfg.Port)
	log.Info().Msgf("API is served at http://localhost%s", cfg.Port)
	log.Info().Msgf("Swagger UI is served at http://localhost%s/docs", cfg.Port)

	if err := http.ListenAndServe(cfg.Port, mux); err != nil {
		log.Fatal().Err(err).Msg("Failed to serve HTTP")
	}
}

func setupSwaggerUI(mux *http.ServeMux, swaggerUIDir, swaggerJSONPath string) {
	// Serve the Swagger JSON file
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerJSONPath)
	})

	// Set up Swagger UI
	opts := middleware.SwaggerUIOpts{
		SpecURL: "/swagger.json",
		Path:    "docs",
	}
	sh := middleware.SwaggerUI(opts, nil)
	mux.Handle("/docs", sh)

	// Serve Swagger UI static files
	fs := http.FileServer(http.Dir(swaggerUIDir))
	mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui", fs))
}
