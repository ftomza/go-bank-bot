package bot_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ftomza/go-bank-bot/pkg/bot"
)

func TestSession(t *testing.T) {

	session := bot.NewSession(context.Background(), bot.NewNextStep(func(ctx context.Context, sess *bot.Session) error {
		sess.AddValue("test1", "test")
		return nil
	}, bot.NewConditionalStep(func(ctx context.Context, sess *bot.Session) (bool, error) {
		_, ok := sess.Value("test").(string)
		return ok, nil
	}, bot.NewNextStep(func(ctx context.Context, sess *bot.Session) error {
		t.Log(sess.Value("test").(string))
		return nil
	}, nil), bot.NewNextStep(func(ctx context.Context, sess *bot.Session) error {
		t.Log("Bad!!!")
		return nil
	}, nil))))

	ctx := context.Background()

	assert.NoError(t, session.Run(ctx))

	assert.NoError(t, session.Run(ctx))

	assert.NoError(t, session.Run(ctx))

}
