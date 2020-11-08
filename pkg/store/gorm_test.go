package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/ftomza/go-bank-bot/domain"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type GormUserRepositoryTestSuite struct {
	suite.Suite
	Ctx  context.Context
	Mock sqlmock.Sqlmock
	DB   *gorm.DB
	Repo domain.UserRepository
}

func (suite *GormUserRepositoryTestSuite) SetupTest() {
	var (
		err error
	)

	suite.DB, err = gorm.Open(sqlite.Open("file:ent?mode=memory&cache=shared&_fk=1"), &gorm.Config{})
	suite.NoError(err)

	suite.DB = suite.DB.Debug()

	suite.Repo = NewGormUserRepository(suite.DB)

	suite.Ctx = context.Background()

	suite.NoError(suite.Repo.Migration(suite.Ctx))
}

func Test_EntWalletRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(GormUserRepositoryTestSuite))
}

func (suite *GormUserRepositoryTestSuite) TearDownTest() {
}

func (suite *GormUserRepositoryTestSuite) Test_GormUserRepository_Store() {
	suite.Run("ok", func() {
		item := domain.User{
			ID:        1,
			BotUserID: 13,
			TrxPatterns: TrxPatterns{
				{Pattern: "ptr1"},
				{Pattern: "ptr2"},
			},
		}
		err := suite.Repo.Store(suite.Ctx, &item)
		if suite.NoError(err, "Store") {
			var users []User
			suite.NoError(suite.DB.Model(&User{}).Where(&User{Model: gorm.Model{ID: item.ID}}).Find(&users).Error)
			if suite.Len(users, 1) {
				suite.Equal(users[0].ID, item.ID)
				suite.Equal(users[0].BotUserID, item.BotUserID)
				suite.Equal(users[0].TrxPatterns, TrxPatterns(item.TrxPatterns))
			}
		}
	})

	suite.Run("fail unique", func() {
		item := domain.User{
			ID:        2,
			BotUserID: 13,
		}
		err := suite.Repo.Store(suite.Ctx, &item)
		suite.EqualError(err, "UNIQUE constraint failed: users.bot_user_id", "Store")
	})

}

func (suite *GormUserRepositoryTestSuite) Test_GormUserRepository_Get() {
	suite.Run("ok", func() {
		item := domain.User{
			ID:        1,
			BotUserID: 13,
		}
		err := suite.Repo.Store(suite.Ctx, &item)
		if suite.NoError(err, "Store") {
			user, err := suite.Repo.Get(suite.Ctx, 1)
			if suite.NoError(err) {
				suite.Equal(user.ID, item.ID)
				suite.Equal(user.BotUserID, item.BotUserID)
			}
		}
	})
}

func (suite *GormUserRepositoryTestSuite) Test_GormUserRepository_GetByBotUserID() {
	suite.Run("ok", func() {
		item := domain.User{
			ID:        1,
			BotUserID: 13,
		}
		err := suite.Repo.Store(suite.Ctx, &item)
		if suite.NoError(err, "Store") {
			user, err := suite.Repo.GetByBotUserID(suite.Ctx, 13)
			if suite.NoError(err) {
				suite.Equal(user.ID, item.ID)
				suite.Equal(user.BotUserID, item.BotUserID)
			}
		}
	})
}

func (suite *GormUserRepositoryTestSuite) Test_GormUserRepository_Update() {
	suite.Run("ok", func() {
		item := domain.User{
			ID:        1,
			BotUserID: 13,
		}
		err := suite.Repo.Store(suite.Ctx, &item)
		if !suite.NoError(err, "Store") {
			return
		}
		item.SheetID = "test"
		if suite.NoError(suite.Repo.Update(suite.Ctx, &item)) {
			user, err := suite.Repo.Get(suite.Ctx, 1)
			if suite.NoError(err) {
				suite.Equal(user.ID, item.ID)
				suite.Equal(user.BotUserID, item.BotUserID)
				suite.Equal(user.SheetID, item.SheetID)
			}
		}
	})

	suite.Run("fail", func() {
		item := domain.User{
			ID:        1,
			BotUserID: 13,
		}
		item.ID = 2
		item.SheetID = "test"
		err := suite.Repo.Update(suite.Ctx, &item)
		suite.EqualError(err, "record not found", "Store")
	})
}
