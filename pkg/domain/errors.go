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

var (
	ErrCacheMiss          = errors.New("cache miss")
	ErrCacheConnection    = errors.New("cache connection error")
	ErrCacheSerialization = errors.New("cache serialization error")
)

var (
	ErrInvalidScheduledTime         = errors.New("scheduled time must be in the future")
	ErrInvalidBatchItems            = errors.New("batch must contain at least one item")
	ErrBatchSizeExceeded            = errors.New("batch size cannot exceed 1000 items")
	ErrInvalidLimit                 = errors.New("invalid transaction limit")
	ErrTransactionLimitExceeded     = errors.New("transaction limit exceeded")
	ErrDailyLimitExceeded           = errors.New("daily transaction limit exceeded")
	ErrDailyCountExceeded           = errors.New("daily transaction count exceeded")
	ErrScheduledTransactionNotFound = errors.New("scheduled transaction not found")
	ErrBatchTransactionNotFound     = errors.New("batch transaction not found")
	ErrCurrencyNotSupported         = errors.New("currency not supported")
	ErrExchangeRateNotFound         = errors.New("exchange rate not found")
)
