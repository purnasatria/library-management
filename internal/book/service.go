package book

import (
	"context"
	"database/sql"
	"fmt"

	author_pb "github.com/purnasatria/library-management/api/gen/author"
	pb "github.com/purnasatria/library-management/api/gen/book"
	category_pb "github.com/purnasatria/library-management/api/gen/category"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	pb.UnimplementedBookServiceServer
	repo            *Repository
	authorService   author_pb.AuthorServiceClient
	categoryService category_pb.CategoryServiceClient
}

func NewService(repo *Repository, authorService author_pb.AuthorServiceClient, categoryService category_pb.CategoryServiceClient) *Service {
	return &Service{
		repo:            repo,
		authorService:   authorService,
		categoryService: categoryService,
	}
}

func (s *Service) CreateBook(ctx context.Context, req *pb.CreateBookRequest) (*pb.BookResponse, error) {
	var book *Book
	err := s.repo.WithTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		book = &Book{
			Title:           req.Title,
			AuthorID:        req.AuthorId,
			ISBN:            req.Isbn,
			PublicationYear: int(req.PublicationYear),
			Publisher:       req.Publisher,
			Description:     req.Description,
			TotalCopies:     int(req.TotalCopies),
			AvailableCopies: int(req.TotalCopies),
		}

		err = s.repo.CreateBook(ctx, tx, book)
		if err != nil {
			return fmt.Errorf("failed to create book: %w", err)
		}

		_, err = s.categoryService.BulkAddItemToCategories(ctx, &category_pb.BulkAddItemToCategoriesRequest{
			ItemId:      book.ID,
			ItemType:    "book",
			CategoryIds: req.CategoryIds,
		})
		if err != nil {
			return fmt.Errorf("failed to add categories: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create book: %v", err)
	}

	return s.bookToProto(ctx, book)
}

func (s *Service) GetBook(ctx context.Context, req *pb.GetBookRequest) (*pb.BookResponse, error) {
	book, err := s.repo.GetBook(ctx, req.Id)
	if err != nil {
		if err == ErrBookNotFound {
			return nil, status.Errorf(codes.NotFound, "book not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get book: %v", err)
	}

	return s.bookToProto(ctx, book)
}

func (s *Service) UpdateBook(ctx context.Context, req *pb.UpdateBookRequest) (*pb.BookResponse, error) {
	var book *Book
	err := s.repo.WithTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		book, err = s.repo.GetBook(ctx, req.Id)
		if err != nil {
			return fmt.Errorf("failed to get book: %w", err)
		}

		book.Title = req.Title
		book.AuthorID = req.AuthorId
		book.ISBN = req.Isbn
		book.PublicationYear = int(req.PublicationYear)
		book.Publisher = req.Publisher
		book.Description = req.Description
		book.TotalCopies = int(req.TotalCopies)

		err = s.repo.UpdateBook(ctx, tx, book)
		if err != nil {
			return fmt.Errorf("failed to update book: %w", err)
		}

		_, err = s.categoryService.UpdateItemCategories(ctx, &category_pb.UpdateItemCategoriesRequest{
			ItemId:      book.ID,
			ItemType:    "book",
			CategoryIds: req.CategoryIds,
		})
		if err != nil {
			return fmt.Errorf("failed to update categories: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update book: %v", err)
	}

	return s.bookToProto(ctx, book)
}

func (s *Service) DeleteBook(ctx context.Context, req *pb.DeleteBookRequest) (*pb.DeleteBookResponse, error) {
	err := s.repo.WithTransaction(ctx, func(tx *sql.Tx) error {
		err := s.repo.DeleteBook(ctx, tx, req.Id)
		if err != nil {
			return fmt.Errorf("failed to delete book: %w", err)
		}

		_, err = s.categoryService.UpdateItemCategories(ctx, &category_pb.UpdateItemCategoriesRequest{
			ItemId:      req.Id,
			ItemType:    "book",
			CategoryIds: []string{}, // Empty array to remove all categories
		})
		if err != nil {
			return fmt.Errorf("failed to remove categories: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete book: %v", err)
	}

	return &pb.DeleteBookResponse{Success: true}, nil
}

func (s *Service) ListBooks(ctx context.Context, req *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
	params := ListBooksParams{
		Page:                 int(req.Page),
		PageSize:             int(req.PageSize),
		TitleQuery:           req.TitleQuery,
		AuthorQuery:          req.AuthorQuery,
		ISBNQuery:            req.IsbnQuery,
		PublicationYearStart: int(req.PublicationYearStart),
		PublicationYearEnd:   int(req.PublicationYearEnd),
		PublisherQuery:       req.PublisherQuery,
		AvailableOnly:        req.AvailableOnly,
		SortBy:               req.SortBy.String(),
		SortDesc:             req.SortDesc,
	}

	books, total, err := s.repo.ListBooks(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list books: %v", err)
	}

	pbBooks := make([]*pb.BookSummary, len(books))
	for i, book := range books {
		pbBook, err := s.bookSummaryToProto(ctx, book)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert book to proto: %v", err)
		}
		pbBooks[i] = pbBook
	}

	return &pb.ListBooksResponse{
		Books: pbBooks,
		Total: int32(total),
	}, nil
}

func (s *Service) BorrowBook(ctx context.Context, req *pb.BorrowBookRequest) (*pb.BorrowBookResponse, error) {
	var transactionID string
	err := s.repo.WithTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		transactionID, err = s.repo.BorrowBook(ctx, tx, req.Id, req.UserId)
		if err != nil {
			return fmt.Errorf("failed to borrow book: %w", err)
		}
		return nil
	})
	if err != nil {
		if err == ErrNoAvailableCopies {
			return nil, status.Errorf(codes.FailedPrecondition, "no available copies")
		}
		return nil, status.Errorf(codes.Internal, "failed to borrow book: %v", err)
	}

	return &pb.BorrowBookResponse{
		Success:       true,
		TransactionId: transactionID,
	}, nil
}

func (s *Service) ReturnBook(ctx context.Context, req *pb.ReturnBookRequest) (*pb.ReturnBookResponse, error) {
	err := s.repo.WithTransaction(ctx, func(tx *sql.Tx) error {
		err := s.repo.ReturnBook(ctx, tx, req.Id, req.UserId, req.TransactionId)
		if err != nil {
			return fmt.Errorf("failed to return book: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to return book: %v", err)
	}

	return &pb.ReturnBookResponse{Success: true}, nil
}

func (s *Service) GetBookRecommendations(ctx context.Context, req *pb.GetBookRecommendationsRequest) (*pb.GetBookRecommendationsResponse, error) {
	// Get categories for the book
	categories, err := s.categoryService.GetItemCategories(ctx, &category_pb.GetItemCategoriesRequest{
		ItemId:   req.Id,
		ItemType: "book",
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get book categories: %v", err)
	}

	categoryIds := make([]string, len(categories.Categories))
	for i, category := range categories.Categories {
		categoryIds[i] = category.Id
	}

	// Get books with related categories
	relatedBooks, err := s.categoryService.GetItemsByCategories(ctx, &category_pb.GetItemsByCategoriesRequest{
		CategoryIds: categoryIds,
		ItemType:    "book",
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get related books: %v", err)
	}

	recommendations, err := s.repo.GetBookRecommendations(ctx, req.Id, relatedBooks.ItemIds, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get book recommendations: %v", err)
	}

	pbRecommendations := make([]*pb.BookSummary, len(recommendations))
	for i, book := range recommendations {
		pbBook, err := s.bookSummaryToProto(ctx, book)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert book to proto: %v", err)
		}
		pbRecommendations[i] = pbBook
	}

	return &pb.GetBookRecommendationsResponse{
		Recommendations: pbRecommendations,
	}, nil
}

func (s *Service) bookToProto(ctx context.Context, book *Book) (*pb.BookResponse, error) {
	author, err := s.authorService.GetAuthor(ctx, &author_pb.GetAuthorRequest{Id: book.AuthorID})
	if err != nil {
		return nil, fmt.Errorf("failed to get author: %w", err)
	}

	categories, err := s.categoryService.GetItemCategories(ctx, &category_pb.GetItemCategoriesRequest{
		ItemId:   book.ID,
		ItemType: "book",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	categoriesSummary := categoriesToSummaries(categories.Categories)

	return &pb.BookResponse{
		Book: &pb.Book{
			Id:              book.ID,
			Title:           book.Title,
			Author:          author.Author,
			Isbn:            book.ISBN,
			PublicationYear: int32(book.PublicationYear),
			Publisher:       book.Publisher,
			Description:     book.Description,
			TotalCopies:     int32(book.TotalCopies),
			AvailableCopies: int32(book.AvailableCopies),
			Categories:      categoriesSummary,
			CreatedAt:       timestamppb.New(book.CreatedAt),
			UpdatedAt:       timestamppb.New(book.UpdatedAt),
		},
	}, nil
}

func (s *Service) bookSummaryToProto(ctx context.Context, book *Book) (*pb.BookSummary, error) {
	author, err := s.authorService.GetAuthor(ctx, &author_pb.GetAuthorRequest{Id: book.AuthorID})
	if err != nil {
		return nil, fmt.Errorf("failed to get author: %w", err)
	}

	categories, err := s.categoryService.GetItemCategories(ctx, &category_pb.GetItemCategoriesRequest{
		ItemId:   book.ID,
		ItemType: "book",
	})

	categoriesSummary := categoriesToSummaries(categories.Categories)

	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return &pb.BookSummary{
		Id:    book.ID,
		Title: book.Title,
		Author: &pb.AuthorSummary{
			Id:   author.Author.Id,
			Name: author.Author.Name,
		},
		Isbn:            book.ISBN,
		PublicationYear: int32(book.PublicationYear),
		Publisher:       book.Publisher,
		TotalCopies:     int32(book.TotalCopies),
		AvailableCopies: int32(book.AvailableCopies),
		Categories:      categoriesSummary,
		CreatedAt:       timestamppb.New(book.CreatedAt),
		UpdatedAt:       timestamppb.New(book.UpdatedAt),
	}, nil
}

func categoriesToSummaries(categories []*category_pb.Category) []*pb.CategorySummary {
	categorySummaries := make([]*pb.CategorySummary, len(categories))
	for i, category := range categories {
		categorySummaries[i] = &pb.CategorySummary{
			Id:   category.Id,
			Name: category.Name,
		}
	}
	return categorySummaries
}
