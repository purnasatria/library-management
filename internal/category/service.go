package category

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/purnasatria/library-management/api/gen/category"
	"github.com/rs/zerolog/log"
)

type Service struct {
	pb.UnimplementedCategoryServiceServer
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CategoryResponse, error) {
	category := &Category{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreateCategory(category); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create category: %v", err)
	}

	return s.categoryToProto(category)
}

func (s *Service) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.CategoryResponse, error) {
	category, err := s.repo.GetCategory(req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "category not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get category: %v", err)
	}

	return s.categoryToProto(category)
}

func (s *Service) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.CategoryResponse, error) {
	category, err := s.repo.GetCategory(req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "category not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get category: %v", err)
	}

	category.Name = req.Name
	category.Description = req.Description

	if err := s.repo.UpdateCategory(category); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update category: %v", err)
	}

	return s.categoryToProto(category)
}

func (s *Service) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	if err := s.repo.DeleteCategory(req.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "category not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete category: %v", err)
	}

	return &pb.DeleteCategoryResponse{Success: true}, nil
}

func (s *Service) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	offset := int(req.Page-1) * int(req.PageSize)
	limit := int(req.PageSize)

	categories, total, err := s.repo.ListCategories(offset, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list categories: %v", err)
	}

	pbCategories := make([]*pb.Category, len(categories))
	for i, category := range categories {
		pbCategory, err := s.categoryToProto(category)
		if err != nil {
			return nil, err
		}
		pbCategories[i] = pbCategory.Category
	}
	log.Debug().Any("data", pbCategories)

	return &pb.ListCategoriesResponse{
		Categories: pbCategories,
		Total:      int32(total),
	}, nil
}

func (s *Service) UpdateItemCategories(ctx context.Context, req *pb.UpdateItemCategoriesRequest) (*pb.UpdateItemCategoriesResponse, error) {
	added, removed, err := s.repo.UpdateItemCategories(req.ItemId, req.ItemType, req.CategoryIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update item categories: %v", err)
	}

	return &pb.UpdateItemCategoriesResponse{
		Success:            true,
		AddedCategoryIds:   added,
		RemovedCategoryIds: removed,
	}, nil
}

func (s *Service) BulkAddItemToCategories(ctx context.Context, req *pb.BulkAddItemToCategoriesRequest) (*pb.BulkAddItemToCategoriesResponse, error) {
	err := s.repo.BulkAddItemToCategories(req.ItemId, req.ItemType, req.CategoryIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to bulk add item to categories: %v", err)
	}

	return &pb.BulkAddItemToCategoriesResponse{
		Success: true,
	}, nil
}

func (s *Service) GetItemCategories(ctx context.Context, req *pb.GetItemCategoriesRequest) (*pb.GetItemCategoriesResponse, error) {
	categories, err := s.repo.GetItemCategories(req.ItemId, req.ItemType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get item categories: %v", err)
	}

	pbCategories := make([]*pb.Category, len(categories))
	for i, category := range categories {
		pbCategory, err := s.categoryToProto(category)
		if err != nil {
			return nil, err
		}
		pbCategories[i] = pbCategory.Category
	}

	return &pb.GetItemCategoriesResponse{
		Categories: pbCategories,
	}, nil
}

func (s *Service) categoryToProto(category *Category) (*pb.CategoryResponse, error) {
	return &pb.CategoryResponse{
		Category: &pb.Category{
			Id:          category.ID,
			Name:        category.Name,
			Description: category.Description,
			CreatedAt:   timestamppb.New(category.CreatedAt),
			UpdatedAt:   timestamppb.New(category.UpdatedAt),
		},
	}, nil
}
