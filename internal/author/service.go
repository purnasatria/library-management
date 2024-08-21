package author

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/purnasatria/library-management/api/gen/author"
)

type Service struct {
	pb.UnimplementedAuthorServiceServer
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAuthor(ctx context.Context, req *pb.CreateAuthorRequest) (*pb.AuthorResponse, error) {
	birthDate := req.BirthDate.AsTime()

	author := &Author{
		Name:      req.Name,
		Biography: req.Biography,
		BirthDate: birthDate,
	}

	if err := s.repo.CreateAuthor(author); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create author: %v", err)
	}

	return s.authorToProto(author)
}

func (s *Service) GetAuthor(ctx context.Context, req *pb.GetAuthorRequest) (*pb.AuthorResponse, error) {
	author, err := s.repo.GetAuthor(req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "author not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get author: %v", err)
	}

	return s.authorToProto(author)
}

func (s *Service) UpdateAuthor(ctx context.Context, req *pb.UpdateAuthorRequest) (*pb.AuthorResponse, error) {
	author, err := s.repo.GetAuthor(req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "author not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get author: %v", err)
	}

	author.Name = req.Name
	author.Biography = req.Biography
	author.BirthDate = req.BirthDate.AsTime()

	if err := s.repo.UpdateAuthor(author); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update author: %v", err)
	}

	return s.authorToProto(author)
}

func (s *Service) DeleteAuthor(ctx context.Context, req *pb.DeleteAuthorRequest) (*pb.DeleteAuthorResponse, error) {
	if err := s.repo.DeleteAuthor(req.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "author not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete author: %v", err)
	}

	return &pb.DeleteAuthorResponse{Success: true}, nil
}

func (s *Service) ListAuthors(ctx context.Context, req *pb.ListAuthorsRequest) (*pb.ListAuthorsResponse, error) {
	offset := int(req.Page-1) * int(req.PageSize)
	limit := int(req.PageSize)

	authors, total, err := s.repo.ListAuthors(offset, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list authors: %v", err)
	}

	pbAuthors := make([]*pb.Author, len(authors))
	for i, author := range authors {
		pbAuthor, err := s.authorToProto(author)
		if err != nil {
			return nil, err
		}
		pbAuthors[i] = pbAuthor.Author
	}

	return &pb.ListAuthorsResponse{
		Authors: pbAuthors,
		Total:   int32(total),
	}, nil
}

func (s *Service) authorToProto(author *Author) (*pb.AuthorResponse, error) {
	return &pb.AuthorResponse{
		Author: &pb.Author{
			Id:        author.ID,
			Name:      author.Name,
			Biography: author.Biography,
			BirthDate: timestamppb.New(author.BirthDate),
			CreatedAt: timestamppb.New(author.CreatedAt),
			UpdatedAt: timestamppb.New(author.UpdatedAt),
		},
	}, nil
}
