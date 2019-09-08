package rocket

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"

	"github.com/go-joe/joe"
	"github.com/go-joe/joe/joetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Dummy ID Generator
type dummygen struct{}

func (d *dummygen) Seed(i int64) {}
func (d *dummygen) ID() string {
	return "1"
}

var dummyUser = &models.User{
	ID:       "123",
	UserName: "dummy_user",
}

var botUser = &models.User{
	ID:           "testID",
	Name:         "Test Name",
	UserName:     "testname",
	Status:       "",
	Token:        "123abc",
	TokenExpires: 0,
}

var dummyRoom = models.Channel{
	ID:   "room0",
	Name: "Room 0",
	Type: "p",
}

var dummyDM = models.Channel{
	ID:   "dm0",
	Name: "DM",
	Type: "d",
}

// compile time test to check if we are implementing the interface.
var _ joe.Adapter = new(BotAdapter)

func newTestAdapter(t *testing.T) (*BotAdapter, *mockRocket) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	client := new(mockRocket)

	creds := &models.UserCredentials{
		Email:    "test@email.com",
		Password: "123",
		Name:     "",
	}
	loginResp := botUser
	client.On("Login", creds).Return(loginResp, nil)
	u, _ := url.Parse("https://nowhere")
	conf := Config{
		Email:     "test@email.com",
		Password:  "123",
		ServerURL: u,
		Name:      "Test Name",
		Debug:     false,
		Logger:    logger,
	}

	myChans := &models.Channel{ID: myMsgs}
	client.On("SubscribeToMessageStream", myChans, mock.AnythingOfType("chan models.Message")).Return(nil)

	chList := []models.Channel{
		dummyRoom,
		dummyDM,
	}
	client.On("GetChannelsIn").Return(chList, nil)

	a, err := newAdapter(ctx, client, conf, &dummygen{})
	require.NoError(t, err)

	return a, client
}

func TestAdapter_IgnoreNormalMessages(t *testing.T) {
	brain := joetest.NewBrain(t)
	a, _ := newTestAdapter(t)

	done := make(chan bool)
	go func() {
		a.handleRocketMessages(brain.Brain)
		done <- true
	}()

	a.messages <- models.Message{
		ID:     "0",
		RoomID: dummyRoom.ID,
		Msg:    "Hello",
	}

	close(a.messages)
	<-done
	brain.Finish()

	assert.Empty(t, brain.RecordedEvents())
}

func TestAdapter_DirectMessages(t *testing.T) {
	brain := joetest.NewBrain(t)
	a, _ := newTestAdapter(t)

	done := make(chan bool)
	go func() {
		a.handleRocketMessages(brain.Brain)
		done <- true
	}()

	msg := models.Message{
		ID:     "0",
		RoomID: dummyDM.ID,
		Msg:    "Hello world",
		User:   dummyUser,
	}

	a.messages <- msg
	close(a.messages)
	<-done
	brain.Finish()

	events := brain.RecordedEvents()
	require.NotEmpty(t, events)
	expectedEvt := joe.ReceiveMessageEvent{Text: "Hello world", Channel: dummyDM.ID, Data: msg, AuthorID: dummyUser.ID}
	assert.Equal(t, expectedEvt, events[0])
}

func TestAdapter_MentionBot(t *testing.T) {
	brain := joetest.NewBrain(t)
	a, _ := newTestAdapter(t)

	done := make(chan bool)
	go func() {
		a.handleRocketMessages(brain.Brain)
		done <- true
	}()

	msg := models.Message{
		ID:     "0",
		RoomID: dummyRoom.ID,
		Msg:    fmt.Sprintf("Hey %s!", a.userLink(a.user.UserName)),
		User:   dummyUser,
	}

	a.messages <- msg
	close(a.messages)
	<-done
	brain.Finish()

	events := brain.RecordedEvents()
	require.NotEmpty(t, events)
	expectedEvt := joe.ReceiveMessageEvent{Text: msg.Msg, Channel: dummyRoom.ID, AuthorID: dummyUser.ID, Data: msg}
	assert.Equal(t, expectedEvt, events[0])
}

func TestAdapter_MentionBotPrefix(t *testing.T) {
	brain := joetest.NewBrain(t)
	a, _ := newTestAdapter(t)

	done := make(chan bool)
	go func() {
		a.handleRocketMessages(brain.Brain)
		done <- true
	}()

	msg := models.Message{
		ID:     "0",
		RoomID: dummyRoom.ID,
		Msg:    fmt.Sprintf("%s PING", a.userLink(a.user.UserName)),
		User:   dummyUser,
	}

	a.messages <- msg
	close(a.messages)
	<-done
	brain.Finish()

	events := brain.RecordedEvents()
	require.NotEmpty(t, events)
	expectedEvt := joe.ReceiveMessageEvent{Text: "PING", Data: msg, AuthorID: dummyUser.ID, Channel: dummyRoom.I}
	assert.Equal(t, expectedEvt, events[0])
}

func TestAdapter_Send(t *testing.T) {
	a, rocketAPI := newTestAdapter(t)
	rocketAPI.On("SendMessage",
		&models.Message{
			ID:     "1",
			RoomID: dummyRoom.ID,
			Msg:    "Hello World",
			User:   botUser,
		},
	).Return(&models.Message{}, nil)

	err := a.Send("Hello World", dummyRoom.ID)
	require.NoError(t, err)
	rocketAPI.AssertExpectations(t)
}

func TestAdapter_Close(t *testing.T) {
	a, slackAPI := newTestAdapter(t)
	slackAPI.On("Close").Return()

	err := a.Close()
	require.NoError(t, err)
	slackAPI.AssertExpectations(t)
}

type mockRocket struct {
	mock.Mock
}

var _ rocketAPI = new(mockRocket)

func (m *mockRocket) SendMessage(message *models.Message) (msg *models.Message, err error) {
	args := m.Called(message)
	if x := args.Get(0); x != nil {
		msg = x.(*models.Message)
	}

	return msg, args.Error(1)
}

//func (m *mockRocket) ReactToMessage(message *models.Message, reaction string) error
func (m *mockRocket) Login(credentials *models.UserCredentials) (usr *models.User, err error) {
	args := m.Called(credentials)
	if x := args.Get(0); x != nil {
		usr = x.(*models.User)
	}
	return usr, args.Error(1)
}
func (m *mockRocket) SubscribeToMessageStream(channel *models.Channel, msgChannel chan models.Message) error {
	args := m.Called(channel, msgChannel)
	return args.Error(0)
}
func (m *mockRocket) Close() {
	m.Called()
}
func (m *mockRocket) GetChannelsIn() (chs []models.Channel, err error) {
	args := m.Called()
	if x := args.Get(0); x != nil {
		chs = x.([]models.Channel)
	}
	return chs, args.Error(1)
}
