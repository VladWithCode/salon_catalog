package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string `db:"id" json:"id"`
	Fullname string `db:"fullname" json:"fullname"`
	Password string `db:"password" json:"password"`
	Username string `db:"username" json:"username"`
	Role     string `db:"role" json:"role"`
	Email    string `db:"email" json:"email"`
}

func (u *User) ValidatePass(pw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pw))

	if err != nil {
		return err
	}

	return nil
}

func (u *User) HashPass(pw string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)

	return nil
}

const (
	RoleAdmin  string = "admin"
	RoleEditor string = "editor"
	RoleUser   string = "user"
)

type UserDTO struct {
	ID       string `json:"id"`
	Fullname string `json:"fullname"`
	Password string `json:"password"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

func CreateUser(user *User) (string, error) {
	conn, err := GetConn()
	if err != nil {
		return "", err
	}
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	_, err = conn.Exec(
		ctx,
		"INSERT INTO users (id, fullname, password, username, role, email) VALUES ($1, $2, $3, $4, $5, $6)",
		user.ID,
		user.Fullname,
		hashedPassword,
		user.Username,
		user.Role,
		user.Email,
	)

	if err != nil {
		return "", err
	}

	return user.ID, nil
}

func GetUserByID(id string) (*User, error) {
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user User

	err = conn.QueryRow(
		ctx,
		"SELECT * FROM users WHERE id = $1",
		id,
	).Scan(
		&user.ID,
		&user.Fullname,
		&user.Password,
		&user.Username,
		&user.Role,
		&user.Email,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user User

	err = conn.QueryRow(
		ctx,
		"SELECT * FROM users WHERE username = $1",
		username,
	).Scan(
		&user.ID,
		&user.Fullname,
		&user.Password,
		&user.Username,
		&user.Role,
		&user.Email,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUser(user *User) error {
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = conn.Exec(
		ctx,
		"UPDATE users SET name = $1, lastname = $2, password = $3, username = $4, role = $5, email = $6, phone = $7, img = $8 WHERE id = $9",
		user.Fullname,
		user.Password,
		user.Username,
		user.Role,
		user.Email,
	)

	if err != nil {
		return err
	}

	return nil
}

func TxVerifyUserEmail(ctx context.Context, tx pgx.Tx, userID string) error {
	tag, err := tx.Exec(
		ctx,
		"UPDATE users SET email_verified = TRUE WHERE id = $1",
		userID,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no se encontró usuario con id %v", userID)
	}

	return nil
}

func VerifyUserEmail(userID string) error {
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tag, err := conn.Exec(
		ctx,
		"UPDATE users SET email_verified = TRUE WHERE id = $1",
		userID,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no se encontró usuario con id %v", userID)
	}

	return nil
}
