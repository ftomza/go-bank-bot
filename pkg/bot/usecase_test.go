package bot

import (
	"reflect"
	"testing"
	"time"

	"github.com/ftomza/go-bank-bot/domain"
	"github.com/shopspring/decimal"
)

func Test_prepareTransactionOfMessage(t *testing.T) {
	type args struct {
		pattern string
		msg     string
	}
	tests := []struct {
		name    string
		args    args
		want    *domain.Transaction
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				pattern: `^(?P<currency>[A-Z]{3}?) (?P<amount>[0-9\.]+?) is (?P<direction>c?)harged on .*[Cc]ard.*(?P<account>5098?) from (?P<party>.+?) on (?P<date>[0-9\/]{5,}?)\. Combined Avail.Bal is (?P<total>[0-9]+\.[0-9]{2}?).*$`,
				msg:     "AED 1123.33 is charged on Credit Card ending 5098 from FACEBK on 31/10. Combined Avail.Bal is 13274.59. Ref statement for exact amnt.",
			},
			want: &domain.Transaction{
				Account:   "5098",
				Party:     "FACEBK",
				Direction: "c",
				Amount:    decimal.NewFromFloat(1123.33),
				Currency:  "AED",
				Date:      time.Date(2020, 10, 31, 00, 00, 00, 00, time.UTC),
				Total:     decimal.NewFromFloat(13274.59),
				Raw:       "AED 1123.33 is charged on Credit Card ending 5098 from FACEBK on 31/10. Combined Avail.Bal is 13274.59. Ref statement for exact amnt.",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prepareTransactionOfMessage(tt.args.pattern, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareTransactionOfMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareTransactionOfMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
