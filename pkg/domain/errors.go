package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidName        = errors.New("name must not be empty")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrInvalidUsername    = errors.New("username must be at least 3 characters")
)

var (
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrInvalidOperation         = errors.New("invalid operation")
	ErrInvalidTransactionStatus = errors.New("invalid transaction status")
	ErrInvalidState             = errors.New("invalid transaction state")
	ErrTransactionFailed        = errors.New("transaction failed")
)

// Balance errors
var (
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("invalid amount")
)
