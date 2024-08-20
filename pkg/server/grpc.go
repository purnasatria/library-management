package server

import (
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type GRPCServerConfig struct {
	Port              string
	RegisterService   func(*grpc.Server)
	UnaryInterceptors []grpc.UnaryServerInterceptor
}

func RunGRPCServer(cfg GRPCServerConfig) {
	lis, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to listen on port: %v", cfg.Port)
	}

	opts := []grpc.ServerOption{}
	if len(cfg.UnaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(cfg.UnaryInterceptors...))
	}

	s := grpc.NewServer(opts...)
	cfg.RegisterService(s)

	log.Info().Msgf("Starting gRPC server on %s", cfg.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("Failed to serve gRPC")
	}
}
