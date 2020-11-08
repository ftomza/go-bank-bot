package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ftomza/go-bank-bot/pkg/store"

	"github.com/ftomza/go-bank-bot/domain"
	"gopkg.in/tucnak/telebot.v2"
)

var (
	currentMessage = struct{}{}

	endpointForbidden       = "\fforbidden"
	endpointCommandNotFound = "\fcommandNotFound"
)

type commandMessageFn func(c telegramBotCommand, msg *telebot.Message)

type telegramBotCommand struct {
	Name        string
	Command     string
	Description string

	handler interface{}
}

func (c telegramBotCommand) String() string {
	return c.Name
}

func (c telegramBotCommand) BotCommand() telebot.Command {
	return telebot.Command{
		Text:        c.Command,
		Description: c.Description,
	}
}

func (c *telegramBotCommand) AddBotMessageHandle(b *TelegramBot, handler commandMessageFn) {
	c.handler = handler
	b.bot.Handle("/"+c.Command, func(msg *telebot.Message) {
		handler(*c, msg)
	})
}

func (c telegramBotCommand) CallMessageHandler(msg *telebot.Message) {
	c.handler.(commandMessageFn)(c, msg)
}

var (
	startCommand          = telegramBotCommand{Name: "Start", Command: "start", Description: "Start bot"}
	mainCommand           = telegramBotCommand{Name: "Main", Command: "main", Description: "Show main menu"}
	addGoogleTokenCommand = telegramBotCommand{Name: "AddGoogleToken", Command: "addgoogletoken", Description: "Add google token"}
	setSheetCommand       = telegramBotCommand{Name: "SetSheet", Command: "setsheet", Description: "Set Google sheet id for parse data"}
	setSheetListCommand   = telegramBotCommand{Name: "SetSheetList", Command: "setsheetlist", Description: "Set Google sheet list for parse data"}
	setPatternsCommand    = telegramBotCommand{Name: "SetPatterns", Command: "setpatterns", Description: "Set Patterns for parsing input message"}
	cancelCommand         = telegramBotCommand{Name: "Cancel", Command: "cancel", Description: "Cancel current operation"}
)

type sessionBot struct {
	Name    string
	Session Session
}

type TelegramBot struct {
	bot       *telebot.Bot
	userRepo  domain.UserRepository
	trxClient *store.GoogleClient
	sessions  map[int]sessionBot

	startSelector *telebot.ReplyMarkup
}

type TelegramBotMessageEntity telebot.MessageEntity

func (e TelegramBotMessageEntity) IsCommand() bool {
	return e.Type == telebot.EntityCommand
}

type TelegramBotMessage telebot.Message

func (m TelegramBotMessage) Command() string {
	command := m.CommandWithAt()

	if i := strings.Index(command, "@"); i != -1 {
		command = command[:i]
	}

	return command
}

func (m TelegramBotMessage) CommandWithAt() string {
	if !m.IsCommand() {
		return ""
	}
	entity := m.Entities[0]
	return m.Text[1:entity.Length]
}

func (m TelegramBotMessage) IsCommand() bool {
	if m.Entities == nil || len(m.Entities) == 0 {
		return false
	}

	entity := TelegramBotMessageEntity(m.Entities[0])
	return entity.Offset == 0 && entity.IsCommand()
}

func NewPoller(timeoutSec time.Duration) *telebot.MiddlewarePoller {
	poller := &telebot.LongPoller{Timeout: timeoutSec * time.Second}

	middlewarePoller := telebot.NewMiddlewarePoller(poller, func(upd *telebot.Update) bool {
		if upd.Message != nil && !upd.Message.Private() {
			upd.Message.Text = endpointForbidden
		}
		return true
	})
	middlewarePoller = telebot.NewMiddlewarePoller(middlewarePoller, func(upd *telebot.Update) bool {
		if upd.Message != nil {
			msg := TelegramBotMessage(*upd.Message)
			if msg.IsCommand() {
				switch msg.Command() {
				case "main", "start", "addgoogletoken", "setsheet", "setsheetlist", "setpatterns", "cancel":
				default:
					upd.Message.Text = endpointCommandNotFound
				}
			}
		}
		return true
	})

	return middlewarePoller
}

func NewTelegramBot(bot *telebot.Bot, userRepo domain.UserRepository, trxClient *store.GoogleClient) *TelegramBot {

	instance := &TelegramBot{
		bot:       bot,
		userRepo:  userRepo,
		trxClient: trxClient,
		sessions:  map[int]sessionBot{},
	}

	instance.startSelector = instance.newStartSelector(bot)

	bot.Handle(endpointForbidden, instance.forbiddenHandler)
	bot.Handle(endpointCommandNotFound, instance.commandNotFoundHandler)

	startCommand.AddBotMessageHandle(instance, instance.startHandler)
	mainCommand.AddBotMessageHandle(instance, instance.startHandler)

	addGoogleTokenCommand.AddBotMessageHandle(instance, instance.addGoogleTokenHandler)
	setSheetCommand.AddBotMessageHandle(instance, instance.setSheetHandler)
	setSheetListCommand.AddBotMessageHandle(instance, instance.setSheetListHandler)
	setPatternsCommand.AddBotMessageHandle(instance, instance.setPatternsHandler)
	cancelCommand.AddBotMessageHandle(instance, instance.cancelHandler)

	bot.Handle(telebot.OnText, instance.onTextHandler)

	_ = instance.setCommands(
		mainCommand,
		addGoogleTokenCommand,
		setSheetCommand,
		setSheetListCommand,
		setPatternsCommand,
		cancelCommand,
	)

	return instance
}

func (tg *TelegramBot) newStartSelector(bot *telebot.Bot) *telebot.ReplyMarkup {
	selector := &telebot.ReplyMarkup{}
	btnAddGoogleToken := selector.Data("Add Google Token", "addGoogleToken")
	btnSetSheet := selector.Data("Set Sheet ID", "setSheet")
	btnSetSheetList := selector.Data("Set Sheet List", "setSheetList")
	btnSetPatternsList := selector.Data("Set Patterns for parser", "setPatterns")
	selector.Inline(
		selector.Row(btnAddGoogleToken),
		selector.Row(btnSetSheet),
		selector.Row(btnSetSheetList),
		selector.Row(btnSetPatternsList),
	)

	bot.Handle(&btnAddGoogleToken, func(c *telebot.Callback) {
		addGoogleTokenCommand.CallMessageHandler(&telebot.Message{Sender: c.Sender})
	})
	bot.Handle(&btnSetSheet, func(c *telebot.Callback) {
		setSheetCommand.CallMessageHandler(&telebot.Message{Sender: c.Sender})
	})
	bot.Handle(&btnSetSheetList, func(c *telebot.Callback) {
		setSheetListCommand.CallMessageHandler(&telebot.Message{Sender: c.Sender})
	})
	bot.Handle(&btnSetPatternsList, func(c *telebot.Callback) {
		setPatternsCommand.CallMessageHandler(&telebot.Message{Sender: c.Sender})
	})

	return selector
}

func (tg *TelegramBot) setCommands(items ...telegramBotCommand) error {
	var commands []telebot.Command
	for _, v := range items {
		commands = append(commands, v.BotCommand())
	}
	return tg.bot.SetCommands(commands)
}

func (tg *TelegramBot) Start() {
	tg.bot.Start()
}

func (tg *TelegramBot) Stop() {
	tg.bot.Stop()
}

func (tg *TelegramBot) Send(to telebot.Recipient, what interface{}, options ...interface{}) error {
	for {
		if _, err := tg.bot.Send(to, what, options...); err != nil {
			var floodError *telebot.FloodError
			if errors.As(err, &floodError) {
				time.Sleep(time.Duration(floodError.RetryAfter) * time.Second)
				continue
			} else {
				return err
			}
		}
		break
	}
	return nil
}

func (tg *TelegramBot) forbiddenHandler(m *telebot.Message) {
	_ = tg.Send(m.Sender, "Only private message!")
}

func (tg *TelegramBot) commandNotFoundHandler(m *telebot.Message) {
	_ = tg.Send(m.Sender, "Sorry. Command not found! :(")
}

func (tg *TelegramBot) startHandler(_ telegramBotCommand, m *telebot.Message) {
	_ = tg.wrapperErr(m, func() error {
		user, err := tg.GetRepoUser(m.Sender.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		txt := fmt.Sprintf(`
__Settings for: _%s_ __:

\- *Google token*: %s
\- *Sheet ID*: %s
\- *Sheet List*: %s
\- *Patterns*: %s
	`,
			EscapeMarkdown2(m.Sender.Username),
			IfThenElse(user.TokSheet == nil, "🚫", "✔"),
			IfThenElse(user.SheetID == "", "🚫", "✔"),
			IfThenElse(user.ListName == "", "🚫", "✔"),
			IfThenElse(user.TrxPatterns == nil, "🚫", "✔"),
		)

		return tg.Send(m.Sender, txt, tg.startSelector, telebot.ModeMarkdownV2)
	})
}

func EscapeMarkdown2(in string) string {
	for _, v := range []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"} {
		in = strings.Replace(in, v, fmt.Sprintf("\\%s", v), -1)
	}
	return in
}

func (tg *TelegramBot) addGoogleTokenHandler(c telegramBotCommand, m *telebot.Message) {
	_ = tg.Send(m.Sender, tg.trxClient.NewRegistration())
	tg.wrapperSession(m, c.Name, func(cancel context.CancelFunc) Step {
		return NewStep(func(ctx context.Context, sess *Session) error {
			defer cancel()
			return tg.wrapperCtxMessage(ctx, func(msg *telebot.Message) error {
				return tg.wrapperErr(msg, func() error {
					tok, err := tg.trxClient.GetToken(ctx, strings.TrimSpace(msg.Text))
					if err != nil {
						return err
					}
					err = tg.SaveRepoUserGoogleToken(msg.Sender.ID, tok)
					if err != nil {
						return err
					}
					return tg.Send(msg.Sender, "Google token: ✔")
				})
			})
		})
	})
}

func (tg *TelegramBot) setSheetHandler(c telegramBotCommand, m *telebot.Message) {
	_ = tg.Send(m.Sender, "Please set google sheet id")
	tg.wrapperSession(m, c.Name, func(cancel context.CancelFunc) Step {
		return NewStep(func(ctx context.Context, sess *Session) error {
			defer cancel()
			return tg.wrapperCtxMessage(ctx, func(msg *telebot.Message) error {
				return tg.wrapperErr(msg, func() error {
					err := tg.SaveRepoUserSheet(msg.Sender.ID, strings.TrimSpace(msg.Text))
					if err != nil {
						return err
					}
					return tg.Send(msg.Sender, "Google token: ✔")
				})
			})
		})
	})
}

func (tg *TelegramBot) setSheetListHandler(c telegramBotCommand, m *telebot.Message) {
	_ = tg.Send(m.Sender, "Please set google sheet list")
	tg.wrapperSession(m, c.Name, func(cancel context.CancelFunc) Step {
		return NewStep(func(ctx context.Context, sess *Session) error {
			defer cancel()
			return tg.wrapperCtxMessage(ctx, func(msg *telebot.Message) error {
				return tg.wrapperErr(msg, func() error {
					err := tg.SaveRepoUserSheetList(msg.Sender.ID, strings.TrimSpace(msg.Text))
					if err != nil {
						return err
					}
					return tg.Send(msg.Sender, "Sheet ID: ✔")
				})
			})
		})
	})
}
func (tg *TelegramBot) setPatternsHandler(c telegramBotCommand, m *telebot.Message) {
	_ = tg.Send(m.Sender, "Please set patterns")
	tg.wrapperSession(m, c.Name, func(cancel context.CancelFunc) Step {
		return NewStep(func(ctx context.Context, sess *Session) error {
			defer cancel()
			return tg.wrapperCtxMessage(ctx, func(msg *telebot.Message) error {
				return tg.wrapperErr(msg, func() error {
					err := tg.SaveRepoUserPatterns(msg.Sender.ID, strings.Split(strings.TrimSpace(msg.Text), "\n"))
					if err != nil {
						return err
					}
					return tg.Send(msg.Sender, "Sheet List: ✔")
				})
			})
		})
	})
}

func (tg *TelegramBot) cancelHandler(_ telegramBotCommand, m *telebot.Message) {
	sb, ok := tg.sessions[m.Sender.ID]
	if !ok {
		_ = tg.Send(m.Sender, "There is nothing to cancel! :(")
		return
	}
	delete(tg.sessions, m.Sender.ID)
	_ = tg.Send(m.Sender, fmt.Sprintf("The command %s has been cancelled", sb.Name))
}

func (tg *TelegramBot) onTextHandler(m *telebot.Message) {
	_ = tg.runSession(m, NewSession(context.Background(), NewStep(func(ctx context.Context, sess *Session) error {
		return tg.wrapperCtxMessage(ctx, func(msg *telebot.Message) error {
			return tg.wrapperErr(msg, func() error {
				ok, err := tg.ParseAndSaveMessage(msg.Sender.ID, strings.TrimSpace(msg.Text))
				if err != nil {
					return err
				}
				return tg.Send(msg.Sender, IfThenElse(ok, "Message save.", "Message skip."))
			})
		})
	})))
}

func (tg *TelegramBot) runSession(m *telebot.Message, def Session) error {

	ctx := context.WithValue(context.Background(), currentMessage, m)

	sb, ok := tg.sessions[m.Sender.ID]
	if ok {
		err := sb.Session.Run(ctx)
		if sb.Session.step == nil || err != nil {
			delete(tg.sessions, m.Sender.ID)
		}
		return err
	}
	return def.Run(ctx)
}

func (tg *TelegramBot) wrapperErr(m *telebot.Message, fn func() error) error {
	if err := fn(); err != nil {
		_ = tg.Send(m.Sender, fmt.Sprintf("Oops, error: %v. Please try again!", err))
		return err
	}
	return nil
}

func (tg *TelegramBot) wrapperCtxMessage(ctx context.Context, fn func(m *telebot.Message) error) error {
	m, ok := ctx.Value(currentMessage).(*telebot.Message)
	if !ok {
		return errors.New("message not found on context")
	}
	return fn(m)
}

func (tg *TelegramBot) wrapperSession(m *telebot.Message, name string, fn func(cancel context.CancelFunc) Step) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	step := fn(cancel)
	tg.sessions[m.Sender.ID] = sessionBot{
		Name:    name,
		Session: NewSession(ctx, step),
	}
}

func IfThenElse(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}
