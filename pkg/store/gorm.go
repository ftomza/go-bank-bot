package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/ftomza/go-bank-bot/domain"
	"gorm.io/gorm"
)

type TrxPatterns []domain.TrxPattern

type User struct {
	gorm.Model

	BotUserID   int    `gorm:"unique"`
	TokSheet    []byte `gorm:"index"`
	SheetID     string `gorm:"index"`
	ListName    string
	TrxPatterns TrxPatterns
}

type DomainUser domain.User

func (u DomainUser) ToUser() User {
	return User{
		Model: gorm.Model{
			ID: u.ID,
		},
		BotUserID:   u.BotUserID,
		TokSheet:    u.TokSheet,
		SheetID:     u.SheetID,
		ListName:    u.ListName,
		TrxPatterns: u.TrxPatterns,
	}
}

func (u User) ToAPIMessage() domain.User {
	return domain.User{
		ID:          u.ID,
		BotUserID:   u.BotUserID,
		TokSheet:    u.TokSheet,
		SheetID:     u.SheetID,
		ListName:    u.ListName,
		TrxPatterns: u.TrxPatterns,
	}
}

func (p *TrxPatterns) Scan(value interface{}) (err error) {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal JSON value: %v", value)
	}
	return json.Unmarshal(bytes, p)
}

func (p TrxPatterns) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (TrxPatterns) GormDataType() string {
	return "string"
}

type gormUserRepository struct {
	db *gorm.DB
}

func (g *gormUserRepository) Migration(_ context.Context) error {
	return g.db.AutoMigrate(&User{})
}

func (g *gormUserRepository) Get(ctx context.Context, id uint) (domain.User, error) {
	item := User{}
	err := g.wrapper(ctx, func(db *gorm.DB) error {
		return db.Where(&User{Model: gorm.Model{ID: id}}).Take(&item).Error
	})
	return item.ToAPIMessage(), err
}

func (g *gormUserRepository) GetByBotUserID(ctx context.Context, uid int) (domain.User, error) {
	item := User{}
	err := g.wrapper(ctx, func(db *gorm.DB) error {
		return db.Where(&User{BotUserID: uid}).Take(&item).Error
	})
	return item.ToAPIMessage(), err
}

func (g *gormUserRepository) Store(ctx context.Context, user *domain.User) error {
	return g.wrapper(ctx, func(db *gorm.DB) error {
		item := DomainUser(*user).ToUser()
		return db.Create(&item).Error
	})
}

func (g *gormUserRepository) Update(ctx context.Context, user *domain.User) error {
	return g.wrapper(ctx, func(db *gorm.DB) error {
		return db.Transaction(func(tx *gorm.DB) error {
			item := DomainUser(*user).ToUser()
			return tx.Take(&User{}, user.ID).
				Updates(&item).Error
		})
	})
}

func (g *gormUserRepository) Delete(ctx context.Context, user *domain.User) error {
	return g.wrapper(ctx, func(db *gorm.DB) error {
		return db.Delete(&User{}, user.ID).Error
	})
}
func (g *gormUserRepository) wrapper(ctx context.Context, fn func(db *gorm.DB) error) error {
	return fn(g.db.WithContext(ctx).Model(&User{}))
}

func NewGormUserRepository(db *gorm.DB) domain.UserRepository {
	return &gormUserRepository{
		db: db,
	}
}
