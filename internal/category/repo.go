package category

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Category struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateCategory(category *Category) error {
	category.ID = uuid.New().String()
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO categories (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, category.ID, category.Name, category.Description, category.CreatedAt, category.UpdatedAt)

	return err
}

func (r *Repository) GetCategory(id string) (*Category, error) {
	category := &Category{}
	err := r.db.QueryRow(`
		SELECT id, name, description, created_at, updated_at
		FROM categories
		WHERE id = $1
	`, id).Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return category, nil
}

func (r *Repository) UpdateCategory(category *Category) error {
	category.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		UPDATE categories
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1
	`, category.ID, category.Name, category.Description, category.UpdatedAt)

	return err
}

func (r *Repository) DeleteCategory(id string) error {
	_, err := r.db.Exec("DELETE FROM categories WHERE id = $1", id)
	return err
}

func (r *Repository) ListCategories(offset, limit int) ([]*Category, int, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, created_at, updated_at
		FROM categories
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		category := &Category{}
		err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		categories = append(categories, category)
	}
	log.Debug().Any("data", categories)

	var total int
	err = r.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

func (r *Repository) UpdateItemCategories(itemID, itemType string, newCategoryIDs []string) (added, removed []string, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	var currentCategoryIDs []string
	rows, err := tx.Query("SELECT category_id FROM category_items WHERE item_id = $1 AND item_type = $2", itemID, itemType)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		currentCategoryIDs = append(currentCategoryIDs, id)
	}

	toAdd := difference(newCategoryIDs, currentCategoryIDs)
	toRemove := difference(currentCategoryIDs, newCategoryIDs)

	if len(toRemove) > 0 {
		_, err = tx.Exec("DELETE FROM category_items WHERE item_id = $1 AND item_type = $2 AND category_id = ANY($3)",
			itemID, itemType, pq.Array(toRemove))
		if err != nil {
			return nil, nil, err
		}
		removed = toRemove
	}

	if len(toAdd) > 0 {
		stmt, err := tx.Prepare(pq.CopyIn("category_items", "id", "category_id", "item_id", "item_type", "created_at"))
		if err != nil {
			return nil, nil, err
		}
		defer stmt.Close()

		now := time.Now()
		for _, categoryID := range toAdd {
			_, err := stmt.Exec(uuid.New().String(), categoryID, itemID, itemType, now)
			if err != nil {
				return nil, nil, err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			return nil, nil, err
		}

		added = toAdd
	}

	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}

	return added, removed, nil
}

func (r *Repository) BulkAddItemToCategories(itemID, itemType string, categoryIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(pq.CopyIn("category_items", "id", "category_id", "item_id", "item_type", "created_at"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, categoryID := range categoryIDs {
		_, err := stmt.Exec(uuid.New().String(), categoryID, itemID, itemType, now)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetItemCategories(itemID, itemType string) ([]*Category, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.name, c.description, c.created_at, c.updated_at
		FROM categories c
		JOIN category_items ci ON c.id = ci.category_id
		WHERE ci.item_id = $1 AND ci.item_type = $2
	`, itemID, itemType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		category := &Category{}
		err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

func (r *Repository) GetItemsByCategories(categoryIDs []string, itemType string) ([]string, error) {
	query := `
		SELECT DISTINCT item_id
		FROM category_items
		WHERE category_id = ANY($1)
		AND item_type = $2
	`

	rows, err := r.db.Query(query, pq.Array(categoryIDs), itemType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itemIDs []string
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return nil, err
		}
		itemIDs = append(itemIDs, itemID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return itemIDs, nil
}

func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
