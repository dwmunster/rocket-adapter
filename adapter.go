// Package rocket implements a rocket.chat adapter for the joe bot library.
package rocket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	rt "github.com/RocketChat/Rocket.Chat.Go.SDK/realtime"
	"github.com/go-joe/joe"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const bufsz = 10
const myMsgs = "__my_messages__"

// IDGen represents the capacity to generate unique ID for each message, as rocket.chat requires.
// ksuids would be great, but that would be an external dependency.
type IDGen interface {
	Seed(int64)
	ID() string
}

// BotAdapter implements a joe.Adapter that reads and writes messages to and
// from Rocket.Chat.
type BotAdapter struct {
	context context.Context
	logger  *zap.Logger
	user    *models.User

	rocket   rocketAPI
	messages chan models.Message

	usersMu sync.RWMutex
	users   map[string]joe.User

	rooms   map[string]models.Channel
	roomsMu sync.RWMutex

	idgen IDGen
}

// Config contains the configuration of a BotAdapter.
type Config struct {
	Email       string
	Password    string
	ServerURL   *url.URL
	botUsername string
	Name        string
	Debug       bool
	Logger      *zap.Logger
}

type rocketAPI interface {
	SendMessage(message *models.Message) (*models.Message, error)
	//ReactToMessage(message *models.Message, reaction string) error
	Login(credentials *models.UserCredentials) (*models.User, error)
	SubscribeToMessageStream(channel *models.Channel, msgChannel chan models.Message) error
	Close()
	GetChannelsIn() ([]models.Channel, error)
}

//Adapter returns a new rocket.chat Adapter as joe.Module.
func Adapter(email, password, serverURL, botUser string, opts ...Option) joe.Module {
	return joe.ModuleFunc(func(joeConf *joe.Config) error {
		conf, err := newConf(email, password, serverURL, botUser, joeConf, opts)
		if err != nil {
			return err
		}

		a, err := NewAdapter(joeConf.Context, conf)
		if err != nil {
			return err
		}

		joeConf.SetAdapter(a)
		return nil
	})
}

func newConf(email, password, serverURL, botUser string, joeConf *joe.Config, opts []Option) (Config, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return Config{}, err
	}
	conf := Config{Email: email, Password: password, ServerURL: u, Name: joeConf.Name, botUsername: botUser}

	for _, opt := range opts {
		err := opt(&conf)
		if err != nil {
			return conf, err
		}
	}

	if conf.Logger == nil {
		conf.Logger = joeConf.Logger("rocket")
	}

	return conf, nil
}

// NewAdapter creates a new *BotAdapter that connects to Rocket.Chat. Note that you
// will usually configure the slack adapter as joe.Module (i.e. using the
// Adapter function of this package).
func NewAdapter(ctx context.Context, conf Config) (*BotAdapter, error) {

	client, err := rt.NewClient(conf.ServerURL, conf.Debug)
	if err != nil {
		return nil, err
	}

	gen := NewRandID()

	return newAdapter(ctx, client, conf, gen)
}

func newAdapter(ctx context.Context, client rocketAPI, conf Config, gen IDGen) (*BotAdapter, error) {
	msgCh := make(chan models.Message, bufsz)

	creds := &models.UserCredentials{
		Email:    conf.Email,
		Password: conf.Password,
	}
	user, err := client.Login(creds)
	if err != nil {
		return nil, errors.Wrap(err, "error logging in")
	}

	if user.UserName == "" {
		user.UserName = conf.botUsername
	}

	a := &BotAdapter{
		rocket:   client,
		context:  ctx,
		logger:   conf.Logger,
		users:    map[string]joe.User{}, // TODO: cache expiration?
		rooms:    map[string]models.Channel{},
		messages: msgCh,
		user:     user,
		idgen:    gen,
	}

	if a.logger == nil {
		a.logger = zap.NewNop()
	}

	a.logger.Info("Connected to rocket.chat realtime API",
		zap.String("url", conf.ServerURL.String()),
		zap.String("username", a.user.UserName),
		zap.String("id", a.user.ID),
	)

	err = a.updateRooms()
	if err != nil {
		return a, err
	}

	channel := &models.Channel{ID: myMsgs}
	err = client.SubscribeToMessageStream(channel, msgCh)

	if err != nil {
		return a, err
	}

	return a, nil
}

// RegisterAt implements the joe.Adapter interface by emitting the slack API
// events to the given brain.
func (a *BotAdapter) RegisterAt(brain *joe.Brain) {
	go a.handleRocketMessages(brain)
}

func (a *BotAdapter) handleRocketMessages(brain *joe.Brain) {
	for msg := range a.messages {
		a.handleMessageEvent(msg, brain)
	}
}

func (a *BotAdapter) handleMessageEvent(msg models.Message, brain *joe.Brain) {
	channel := a.roomByID(msg.RoomID)
	direct := channel.Type == "d"
	// check if we have a DM, or standard channel post
	selfLink := a.userLink(a.user.UserName)
	if !direct && !strings.Contains(msg.Msg, selfLink) {
		// msg not for us!
		return
	}

	text := strings.TrimSpace(strings.TrimPrefix(msg.Msg, selfLink))
	brain.Emit(joe.ReceiveMessageEvent{
		Text:     text,
		Channel:  channel.ID,
		AuthorID: msg.User.ID,
		Data:     msg,
	})
}

func (a *BotAdapter) roomByID(roomID string) models.Channel {
	a.roomsMu.RLock()
	room, ok := a.rooms[roomID]
	a.roomsMu.RUnlock()
	if ok {
		return room
	}

	err := a.updateRooms()
	if err != nil {
		return models.Channel{ID: roomID}
	}

	room, ok = a.rooms[roomID]
	if ok {
		return room
	}
	return models.Channel{ID: roomID}
}

func (a *BotAdapter) updateRooms() error {
	a.roomsMu.Lock()
	defer a.roomsMu.Unlock()
	chs, err := a.rocket.GetChannelsIn()
	if err != nil {
		a.logger.Error("Failed to get list of participating channels.",
			zap.String("error", err.Error()),
		)
		return err
	}

	for _, ch := range chs {
		a.rooms[ch.ID] = ch
	}
	return nil
}

// Send implements joe.Adapter by sending all received text messages to the
// given rocket.chat channel ID.
func (a *BotAdapter) Send(text, channelID string) error {
	a.logger.Info("Sending message to channel",
		zap.String("channel_id", channelID),
		// do not leak actual message content since it might be sensitive
	)

	ch := a.roomByID(channelID)

	msg := a.newMessage(&ch, text)
	_, err := a.rocket.SendMessage(msg)

	return err
}

// Close disconnects the adapter from the rocket.chat API.
func (a *BotAdapter) Close() error {
	a.rocket.Close()
	return nil
}

// userLink takes a username and returns the formatting necessary to link it.
func (a *BotAdapter) userLink(username string) string {
	return fmt.Sprintf("@%s", username)
}

// newMessage creates basic message with an ID, a RoomID, and a Msg
// Takes channel and text
func (a *BotAdapter) newMessage(channel *models.Channel, text string) *models.Message {
	return &models.Message{
		ID:     a.idgen.ID(),
		RoomID: channel.ID,
		Msg:    text,
		User:   a.user,
	}
}
