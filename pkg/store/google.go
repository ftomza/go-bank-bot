package store

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ftomza/go-bank-bot/domain"

	"google.golang.org/api/option"

	"google.golang.org/api/sheets/v4"

	"golang.org/x/oauth2"
)

type GoogleClient struct {
	config *oauth2.Config
}

func NewGoogleClient(config *oauth2.Config) *GoogleClient {
	return &GoogleClient{config: config}
}

func (r *GoogleClient) NewRegistration() string {
	return r.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (r *GoogleClient) GetToken(ctx context.Context, authCode string) ([]byte, error) {
	tok, err := r.config.Exchange(ctx, authCode)
	if err != nil {
		return nil, err
	}
	return json.Marshal(tok)
}

func (r *GoogleClient) Get(ctx context.Context, token []byte) (*http.Client, error) {
	tok := &oauth2.Token{}
	err := json.Unmarshal(token, tok)
	if err != nil {
		return nil, err
	}
	return r.config.Client(ctx, tok), nil
}

type GoogleTransactionRepository struct {
	srv      *sheets.Service
	sheetID  string
	listName string
}

func NewGoogleTransactionRepository(client *http.Client, sheetID, listName string) (domain.TransactionRepository, error) {
	srv, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	if sheetID == "" {
		return nil, errors.New("sheet/google: sheet ID not set")
	}

	return &GoogleTransactionRepository{
		srv:      srv,
		sheetID:  sheetID,
		listName: listName,
	}, nil
}

func (s *GoogleTransactionRepository) Store(ctx context.Context, item *domain.Transaction) error {

	valueInputOption := "RAW"
	insertDataOption := "INSERT_ROWS"
	rb := &sheets.ValueRange{
		Values: [][]interface{}{
			{item.Account, item.Party, item.Direction, item.Amount, item.Currency, item.Date, item.Total, item.Raw},
		},
	}
	resp, err := s.srv.Spreadsheets.Values.
		Append(s.sheetID, s.listName, rb).
		ValueInputOption(valueInputOption).
		InsertDataOption(insertDataOption).
		Context(ctx).
		Do()
	log.Println(resp)
	return err
}
