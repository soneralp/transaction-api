package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrInvalidPassword   = errors.New("password must be at least 8 characters")
	ErrInvalidUsername   = errors.New("username must be at least 3 characters")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type User struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(username, email, password string) (*User, error) {
	user := &User{
		Username:  username,
		Email:     email,
		Password:  password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Validate() error {
	if len(u.Username) < 3 {
		return ErrInvalidUsername
	}

	if len(u.Email) < 5 || !isValidEmail(u.Email) {
		return ErrInvalidEmail
	}

	if len(u.Password) < 8 {
		return ErrInvalidPassword
	}

	return nil
}

func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		Password string `json:"-"`
	}{
		Alias:    (*Alias)(u),
		Password: "",
	})
}

func isValidEmail(email string) bool {
	return len(email) > 0 && email[0] != '@' && email[len(email)-1] != '@'
}
