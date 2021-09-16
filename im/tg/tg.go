package tg

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/sillyGirl/core"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Sender struct {
	Message *tb.Message
	matches [][]string
}

var tg = core.NewBucket("tg")
var b *tb.Bot
var Handler = func(message *tb.Message) {
	core.Senders <- &Sender{
		Message: message,
	}
}

func init() {
	token := tg.Get("token")
	if token == "" {
		logs.Warn("未提供telegram机器人token")
		return
	}
	var err error
	b, err = tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		logs.Warn("监听telegram机器人失败：%v", err)
		return
	}
	b.Handle(tb.OnText, Handler)
	logs.Info("监听telegram机器人")
	b.Start()
}

func (sender *Sender) GetContent() string {
	return sender.Message.Text
}

func (sender *Sender) GetUserID() int {
	return sender.Message.Sender.ID
}

func (sender *Sender) GetChatID() int {
	return int(sender.Message.Chat.ID)
}

func (sender *Sender) GetImType() string {
	return "tg"
}

func (sender *Sender) GetMessageID() int {
	return sender.Message.ID
}

func (sender *Sender) GetUsername() string {
	return sender.Message.Sender.Username
}

func (sender *Sender) IsReply() bool {
	return sender.Message.ReplyTo != nil
}

func (sender *Sender) GetReplySenderUserID() int {
	if !sender.IsReply() {
		return 0
	}
	return sender.Message.ReplyTo.ID
}

func (sender *Sender) GetRawMessage() interface{} {
	return sender.Message
}

func (sender *Sender) SetMatch(ss []string) {
	sender.matches = [][]string{ss}
}
func (sender *Sender) SetAllMatch(ss [][]string) {
	sender.matches = ss
}

func (sender *Sender) GetMatch() []string {
	return sender.matches[0]
}

func (sender *Sender) GetAllMatch() [][]string {
	return sender.matches
}

func (sender *Sender) Get(index ...int) string {

	i := 0
	if len(index) != 0 {
		i = index[0]
	}
	if len(sender.matches) == 0 {
		return ""
	}
	if len(sender.matches[0]) < i+1 {
		return ""
	}
	return sender.matches[0][i]
}

func (sender *Sender) IsAdmin() bool {

	return strings.Contains(tg.Get("masters"), fmt.Sprint(sender.Message.Sender.ID))
}

func (sender *Sender) IsMedia() bool {
	return false
}

func (sender *Sender) Reply(rt interface{}) error {
	var r tb.Recipient
	var options = []interface{}{}
	if !sender.Message.FromGroup() {
		r = sender.Message.Sender
	} else {
		r = sender.Message.Chat
		options = []interface{}{&tb.SendOptions{ReplyTo: sender.Message}}
	}
	var err error
	switch rt.(type) {
	case error:
		_, err = b.Send(r, fmt.Sprintf("%v", rt), options...)
	case []byte:
		_, err = b.Send(r, string(rt.([]byte)), options...)
	case string:
		_, err = b.Send(r, rt.(string), options...)
	case *http.Response:
		_, err = b.SendAlbum(r, tb.Album{&tb.Photo{File: tb.FromReader(rt.(*http.Response).Body)}}, options...)
	}
	if err != nil {
		sender.Reply(err)
	}
	return err
}
