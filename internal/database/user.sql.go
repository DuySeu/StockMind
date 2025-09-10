package database

import (
	"time"
)

type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *service) CreateUser(user *User) error {
	query := `INSERT INTO users (name, email, created_at, updated_at) 
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := s.db.QueryRow(query, user.Name, user.Email, time.Now(), time.Now()).Scan(&user.ID)
	return err
}

func (s *service) GetUsers() ([]*User, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM users ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *service) GetUserByID(id int) (*User, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`

	user := &User{}
	err := s.db.QueryRow(query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

func (s *service) UpdateUser(id int, user *User) error {
	query := `UPDATE users SET name = $1, email = $2, updated_at = $3 WHERE id = $4`

	_, err := s.db.Exec(query, user.Name, user.Email, time.Now(), id)
	return err
}

func (s *service) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := s.db.Exec(query, id)
	return err
}
