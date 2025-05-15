package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	FirstName string    `json:"first_name" gorm:"not null"`
	LastName  string    `json:"last_name" gorm:"not null"`
	Role      Role      `json:"role" gorm:"type:varchar(20);not null;default:'user'"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(email, password, firstName, lastName string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}
	if password == "" {
		return nil, ErrInvalidPassword
	}
	if firstName == "" {
		return nil, ErrInvalidName
	}
	if lastName == "" {
		return nil, ErrInvalidName
	}

	return &User{
		ID:        uuid.New(),
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Role:      RoleUser,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (u *User) Update(firstName, lastName string) error {
	if firstName == "" {
		return ErrInvalidName
	}
	if lastName == "" {
		return ErrInvalidName
	}

	u.FirstName = firstName
	u.LastName = lastName
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) ChangePassword(newPassword string) error {
	if newPassword == "" {
		return ErrInvalidPassword
	}

	u.Password = newPassword
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) Validate() error {
	if u.FirstName == "" {
		return ErrInvalidName
	}
	if u.LastName == "" {
		return ErrInvalidName
	}
	if u.Email == "" {
		return ErrInvalidEmail
	}
	if u.Password == "" {
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

func (u *User) HasRole(role Role) bool {
	return u.Role == role
}
