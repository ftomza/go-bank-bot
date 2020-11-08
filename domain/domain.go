package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type User struct {
	ID          uint         `json:"id"`
	BotUserID   int          `json:"bot_user_id"`
	TokSheet    []byte       `json:"tok_sheet"`
	SheetID     string       `json:"sheet_id"`
	ListName    string       `json:"list_name"`
	TrxPatterns []TrxPattern `json:"trx_patterns"`
}

type TrxPattern struct {
	Pattern string `json:"pattern"`
}

type Transaction struct {
	Account   string
	Party     string
	Direction string
	Amount    decimal.Decimal
	Currency  string
	Date      time.Time
	Total     decimal.Decimal
	Raw       string
}

type UserRepository interface {
	Get(ctx context.Context, id uint) (User, error)
	GetByBotUserID(ctx context.Context, uid int) (User, error)
	Store(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, user *User) error
	Migration(ctx context.Context) error
}

type TransactionRepository interface {
	Store(ctx context.Context, user *Transaction) error
}
