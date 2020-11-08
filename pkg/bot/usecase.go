package bot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/ftomza/go-bank-bot/pkg/store"

	"github.com/ftomza/go-bank-bot/domain"
	"gorm.io/gorm"
)

func (tg *TelegramBot) wrapperRepoUser(userID int, fn func(u domain.User) error) error {
	user, err := tg.GetOrCreateRepoUser(userID)
	if err != nil {
		return err
	}
	return fn(user)
}

func (tg *TelegramBot) wrapperRepoUserAndRepoTrx(userID int, fn func(u domain.User, trx domain.TransactionRepository) error) error {
	user, err := tg.GetRepoUser(userID)
	if err != nil {
		return err
	}
	client, err := tg.trxClient.Get(context.Background(), user.TokSheet)
	if err != nil {
		return err
	}
	trx, err := store.NewGoogleTransactionRepository(client, user.SheetID, user.ListName)
	if err != nil {
		return err
	}
	return fn(user, trx)
}

func (tg *TelegramBot) GetRepoUser(userID int) (domain.User, error) {
	return tg.userRepo.GetByBotUserID(context.Background(), userID)
}

func (tg *TelegramBot) GetOrCreateRepoUser(userID int) (domain.User, error) {
	if user, err := tg.GetRepoUser(userID); err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err = tg.userRepo.Store(context.Background(), &domain.User{
			BotUserID: userID,
		})
		if err != nil {
			return domain.User{}, err
		}
		return tg.GetRepoUser(userID)
	} else {
		return user, err
	}
}

func (tg *TelegramBot) SaveRepoUserGoogleToken(userID int, tok []byte) error {
	return tg.wrapperRepoUser(userID, func(u domain.User) error {
		u.TokSheet = tok
		return tg.userRepo.Update(context.Background(), &u)
	})
}

func (tg *TelegramBot) SaveRepoUserSheet(userID int, sheetID string) error {
	return tg.wrapperRepoUser(userID, func(u domain.User) error {
		u.SheetID = sheetID
		return tg.userRepo.Update(context.Background(), &u)
	})
}

func (tg *TelegramBot) SaveRepoUserSheetList(userID int, listName string) error {
	return tg.wrapperRepoUser(userID, func(u domain.User) error {
		u.ListName = listName
		return tg.userRepo.Update(context.Background(), &u)
	})
}

func (tg *TelegramBot) SaveRepoUserPatterns(userID int, ptrs []string) error {
	return tg.wrapperRepoUser(userID, func(u domain.User) error {
		var trxPatterns []domain.TrxPattern
		for _, v := range ptrs {
			trxPatterns = append(trxPatterns, domain.TrxPattern{Pattern: v})
		}
		u.TrxPatterns = trxPatterns
		return tg.userRepo.Update(context.Background(), &u)
	})
}

func (tg *TelegramBot) ParseAndSaveMessage(userID int, msg string) (bool, error) {
	ok := false
	err := tg.wrapperRepoUserAndRepoTrx(userID, func(u domain.User, trx domain.TransactionRepository) error {
		for _, v := range u.TrxPatterns {
			if trans, err := prepareTransactionOfMessage(v.Pattern, msg); err != nil {
				return err
			} else if trans != nil {
				ok = true
				return trx.Store(context.Background(), trans)
			}
		}
		return nil
	})
	return ok, err
}

func getParamsMsg(regEx, msg string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(msg)

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

func prepareTransactionOfMessage(pattern, msg string) (*domain.Transaction, error) {
	params := getParamsMsg(pattern, msg)

	if len(params) == 0 {
		return nil, nil
	}

	amount, err := decimal.NewFromString(params["amount"])
	if err != nil {
		return nil, err
	}

	total, _ := decimal.NewFromString(params["total"])

	date, err := parseDate(params["date"])
	if err != nil {
		return nil, err
	}

	item := &domain.Transaction{
		Account:   params["account"],
		Party:     params["party"],
		Direction: params["direction"],
		Amount:    amount,
		Currency:  params["currency"],
		Date:      date,
		Total:     total,
		Raw:       msg,
	}

	return item, nil
}

func parseDate(text string) (time.Time, error) {
	if strings.Count(text, "/") == 1 {
		text = fmt.Sprintf("%s/%d", text, time.Now().Year())
	}
	return time.Parse("02/01/2006", text)
}
