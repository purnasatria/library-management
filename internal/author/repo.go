package author

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Author struct {
	ID        string
	Name      string
	Biography string
	BirthDate time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateAuthor(author *Author) error {
	author.ID = uuid.New().String()
	author.CreatedAt = time.Now()
	author.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO authors (id, name, biography, birth_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, author.ID, author.Name, author.Biography, author.BirthDate, author.CreatedAt, author.UpdatedAt)

	return err
}

func (r *Repository) GetAuthor(id string) (*Author, error) {
	author := &Author{}
	err := r.db.QueryRow(`
		SELECT id, name, biography, birth_date, created_at, updated_at
		FROM authors
		WHERE id = $1
	`, id).Scan(&author.ID, &author.Name, &author.Biography, &author.BirthDate, &author.CreatedAt, &author.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return author, nil
}

func (r *Repository) UpdateAuthor(author *Author) error {
	author.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		UPDATE authors
		SET name = $2, biography = $3, birth_date = $4, updated_at = $5
		WHERE id = $1
	`, author.ID, author.Name, author.Biography, author.BirthDate, author.UpdatedAt)

	return err
}

func (r *Repository) DeleteAuthor(id string) error {
	_, err := r.db.Exec("DELETE FROM authors WHERE id = $1", id)
	return err
}

func (r *Repository) ListAuthors(offset, limit int) ([]*Author, int, error) {
	rows, err := r.db.Query(`
		SELECT id, name, biography, birth_date, created_at, updated_at
		FROM authors
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var authors []*Author
	for rows.Next() {
		author := &Author{}
		err := rows.Scan(&author.ID, &author.Name, &author.Biography, &author.BirthDate, &author.CreatedAt, &author.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		authors = append(authors, author)
	}

	var total int
	err = r.db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return authors, total, nil
}
