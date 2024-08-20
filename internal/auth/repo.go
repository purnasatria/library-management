package auth

import (
	"database/sql"
)

type User struct {
	ID       string
	Username string
	Email    string
	Password string
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(user *User) error {
	_, err := r.db.Exec("INSERT INTO users (id, username, email, password) VALUES ($1, $2, $3, $4)",
		user.ID, user.Username, user.Email, user.Password)
	return err
}

func (r *Repository) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow("SELECT id, username, email, password FROM users WHERE username = $1", username).
		Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return user, nil
}
