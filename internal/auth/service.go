package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/purnasatria/library-management/pkg/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/purnasatria/library-management/api/gen/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	pb.UnimplementedAuthServiceServer
	repo *Repository
	jwt  *jwt.JWT
}

func NewService(repo *Repository, jwt *jwt.JWT) *Service {
	return &Service{repo: repo, jwt: jwt}
}

func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Generate a salt and hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "server error: %v", err)
	}

	user := &User{
		ID:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword), // Store the salted and hashed password
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, status.Errorf(codes.Internal, "server error: %v", err)
	}

	return &pb.RegisterResponse{UserId: user.ID}, nil
}

func (s *Service) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.repo.GetUserByUsername(req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Compare the provided password with the stored hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	userID, err := s.jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwt.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	userID, err := s.jwt.ValidateAccessToken(req.Token)
	if err != nil {
		return &pb.VerifyTokenResponse{Valid: false}, nil
	}

	return &pb.VerifyTokenResponse{
		Valid:  true,
		UserId: userID,
	}, nil
}
