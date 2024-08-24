package book

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrBookNotFound        = errors.New("book not found")
	ErrNoAvailableCopies   = errors.New("no available copies")
	ErrTransactionRequired = errors.New("transaction is required")
)

type Book struct {
	ID              string
	Title           string
	AuthorID        string
	ISBN            string
	PublicationYear int
	Publisher       string
	Description     string
	TotalCopies     int
	AvailableCopies int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ListBooksParams struct {
	Page                 int
	PageSize             int
	TitleQuery           string
	AuthorQuery          string
	ISBNQuery            string
	PublicationYearStart int
	PublicationYearEnd   int
	PublisherQuery       string
	AvailableOnly        bool
	SortBy               string
	SortDesc             bool
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateBook(ctx context.Context, tx *sql.Tx, book *Book) error {
	if tx == nil {
		return ErrTransactionRequired
	}

	book.ID = uuid.New().String()
	book.CreatedAt = time.Now()
	book.UpdatedAt = time.Now()
	book.AvailableCopies = book.TotalCopies

	query := `
		INSERT INTO books (id, title, author_id, isbn, publication_year, publisher, description, total_copies, available_copies, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := tx.ExecContext(ctx, query, book.ID, book.Title, book.AuthorID, book.ISBN, book.PublicationYear, book.Publisher, book.Description, book.TotalCopies, book.AvailableCopies, book.CreatedAt, book.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create book: %w", err)
	}

	return nil
}

func (r *Repository) GetBook(ctx context.Context, id string) (*Book, error) {
	query := `
		SELECT id, title, author_id, isbn, publication_year, publisher, description, total_copies, available_copies, created_at, updated_at
		FROM books
		WHERE id = $1
	`

	var book Book
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&book.ID, &book.Title, &book.AuthorID, &book.ISBN, &book.PublicationYear, &book.Publisher,
		&book.Description, &book.TotalCopies, &book.AvailableCopies, &book.CreatedAt, &book.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBookNotFound
		}
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	return &book, nil
}

func (r *Repository) UpdateBook(ctx context.Context, tx *sql.Tx, book *Book) error {
	if tx == nil {
		return ErrTransactionRequired
	}

	book.UpdatedAt = time.Now()

	query := `
		UPDATE books
		SET title = $2, author_id = $3, isbn = $4, publication_year = $5, publisher = $6, 
			description = $7, total_copies = $8, available_copies = $9, updated_at = $10
		WHERE id = $1
	`

	_, err := tx.ExecContext(ctx, query, book.ID, book.Title, book.AuthorID, book.ISBN, book.PublicationYear, book.Publisher, book.Description, book.TotalCopies, book.AvailableCopies, book.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update book: %w", err)
	}

	return nil
}

func (r *Repository) DeleteBook(ctx context.Context, tx *sql.Tx, id string) error {
	if tx == nil {
		return ErrTransactionRequired
	}

	query := "DELETE FROM books WHERE id = $1"
	_, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete book: %w", err)
	}

	return nil
}

func (r *Repository) ListBooks(ctx context.Context, params ListBooksParams) ([]*Book, int, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if params.TitleQuery != "" {
		whereClause += fmt.Sprintf(" AND title ILIKE $%d", argCount)
		args = append(args, "%"+params.TitleQuery+"%")
		argCount++
	}

	if params.AuthorQuery != "" {
		whereClause += fmt.Sprintf(" AND author_id IN (SELECT id FROM authors WHERE name ILIKE $%d)", argCount)
		args = append(args, "%"+params.AuthorQuery+"%")
		argCount++
	}

	if params.ISBNQuery != "" {
		whereClause += fmt.Sprintf(" AND isbn ILIKE $%d", argCount)
		args = append(args, "%"+params.ISBNQuery+"%")
		argCount++
	}

	if params.PublicationYearStart != 0 {
		whereClause += fmt.Sprintf(" AND publication_year >= $%d", argCount)
		args = append(args, params.PublicationYearStart)
		argCount++
	}

	if params.PublicationYearEnd != 0 {
		whereClause += fmt.Sprintf(" AND publication_year <= $%d", argCount)
		args = append(args, params.PublicationYearEnd)
		argCount++
	}

	if params.PublisherQuery != "" {
		whereClause += fmt.Sprintf(" AND publisher ILIKE $%d", argCount)
		args = append(args, "%"+params.PublisherQuery+"%")
		argCount++
	}

	if params.AvailableOnly {
		whereClause += " AND available_copies > 0"
	}

	// Count total matching books
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM books %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count books: %w", err)
	}

	// Prepare main query
	query := fmt.Sprintf(`
		SELECT id, title, author_id, isbn, publication_year, publisher, description, total_copies, available_copies, created_at, updated_at
		FROM books
		%s
	`, whereClause)

	// Add ORDER BY clause
	switch params.SortBy {
	case "title":
		query += " ORDER BY title"
	case "author":
		query += " ORDER BY author_id"
	case "publication_year":
		query += " ORDER BY publication_year"
	default:
		query += " ORDER BY created_at"
	}

	if params.SortDesc {
		query += " DESC"
	} else {
		query += " ASC"
	}

	// Add LIMIT and OFFSET
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query books: %w", err)
	}
	defer rows.Close()

	var books []*Book
	for rows.Next() {
		var book Book
		err := rows.Scan(
			&book.ID, &book.Title, &book.AuthorID, &book.ISBN, &book.PublicationYear, &book.Publisher,
			&book.Description, &book.TotalCopies, &book.AvailableCopies, &book.CreatedAt, &book.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan book row: %w", err)
		}
		books = append(books, &book)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after scanning books: %w", err)
	}

	return books, total, nil
}

func (r *Repository) BorrowBook(ctx context.Context, tx *sql.Tx, bookID, userID string) (string, error) {
	if tx == nil {
		return "", ErrTransactionRequired
	}

	query := `
		UPDATE books
		SET available_copies = available_copies - 1
		WHERE id = $1 AND available_copies > 0
		RETURNING available_copies
	`

	var availableCopies int
	err := tx.QueryRowContext(ctx, query, bookID).Scan(&availableCopies)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNoAvailableCopies
		}
		return "", fmt.Errorf("failed to update book copies: %w", err)
	}

	transactionID := uuid.New().String()
	insertQuery := `
		INSERT INTO book_transactions (id, book_id, user_id, transaction_type, transaction_date)
		VALUES ($1, $2, $3, 'borrow', $4)
	`

	_, err = tx.ExecContext(ctx, insertQuery, transactionID, bookID, userID, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to insert borrow transaction: %w", err)
	}

	return transactionID, nil
}

func (r *Repository) ReturnBook(ctx context.Context, tx *sql.Tx, bookID, userID, transactionID string) error {
	if tx == nil {
		return ErrTransactionRequired
	}

	query := `
		UPDATE books
		SET available_copies = available_copies + 1
		WHERE id = $1
		RETURNING available_copies
	`

	var availableCopies int
	err := tx.QueryRowContext(ctx, query, bookID).Scan(&availableCopies)
	if err != nil {
		return fmt.Errorf("failed to update book copies: %w", err)
	}

	insertQuery := `
		INSERT INTO book_transactions (id, book_id, user_id, transaction_type, transaction_date)
		VALUES ($1, $2, $3, 'return', $4)
	`

	_, err = tx.ExecContext(ctx, insertQuery, uuid.New().String(), bookID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert return transaction: %w", err)
	}

	return nil
}

func (r *Repository) GetBookRecommendations(ctx context.Context, bookID string, relatedBookIDs []string, limit int) ([]*Book, error) {
	query := `
		WITH popular_books AS (
			SELECT book_id
			FROM book_transactions
			WHERE transaction_type = 'borrow'
			GROUP BY book_id
			ORDER BY COUNT(*) DESC
			LIMIT 50
		)
		SELECT b.id, b.title, b.author_id, b.isbn, b.publication_year, b.publisher, 
			   b.description, b.total_copies, b.available_copies, b.created_at, b.updated_at,
			   COUNT(bt.id) as borrow_count
		FROM books b
		LEFT JOIN book_transactions bt ON b.id = bt.book_id AND bt.transaction_type = 'borrow'
		WHERE b.id != $1 
		  AND (b.author_id = (SELECT author_id FROM books WHERE id = $1)
			   OR b.id = ANY($2)
			   OR b.id IN (SELECT book_id FROM popular_books))
		GROUP BY b.id
		ORDER BY 
			CASE 
				WHEN b.author_id = (SELECT author_id FROM books WHERE id = $1) THEN 1
				WHEN b.id = ANY($2) THEN 2
				ELSE 3
			END,
			borrow_count DESC,
			b.publication_year DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, bookID, pq.Array(relatedBookIDs), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query book recommendations: %w", err)
	}
	defer rows.Close()

	var recommendations []*Book
	for rows.Next() {
		var book Book
		var borrowCount int
		err := rows.Scan(
			&book.ID, &book.Title, &book.AuthorID, &book.ISBN, &book.PublicationYear, &book.Publisher,
			&book.Description, &book.TotalCopies, &book.AvailableCopies, &book.CreatedAt, &book.UpdatedAt,
			&borrowCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommendation row: %w", err)
		}
		recommendations = append(recommendations, &book)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after scanning recommendations: %w", err)
	}

	return recommendations, nil
}

func (r *Repository) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
