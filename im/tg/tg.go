package tg

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/astaxie/beego/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/sillyGirl/core"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Sender struct {
	Message  *tb.Message
	matches  [][]string
	Duration *time.Duration
	deleted  bool
	reply    *tb.Message
	goon     bool
}

var tg = core.NewBucket("tg")
var b *tb.Bot
var Handler = func(message *tb.Message) {
	if message.FromGroup() {
		if groupCode := tg.GetInt("groupCode"); groupCode != 0 && groupCode != int(message.Chat.ID) {
			return
		}
	}
	core.Senders <- &Sender{
		Message: message,
	}
}

func init() {
	go func() {
		token := tg.Get("token")
		if token == "" {
			logs.Warn("未提供telegram机器人token")
			return
		}
		var err error
		b, err = tb.NewBot(tb.Settings{
			URL:    tg.Get("url"),
			Token:  token,
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
			// ParseMode: tb.ModeMarkdownV2,
		})

		if err != nil {
			logs.Warn("监听telegram机器人失败：%v", err)
			return
		}
		core.Pushs["tg"] = func(i int, s string) {
			b.Send(&tb.User{ID: i}, s)
		}
		core.GroupPushs["tg"] = func(i, j int, s string) {
			paths := []string{}
			ct := &tb.Chat{ID: int64(i)}
			for _, v := range regexp.MustCompile(`\[CQ:image,file=([^\[\]]+)\]`).FindAllStringSubmatch(s, -1) {
				paths = append(paths, core.ExecPath+"/data/images/"+v[1])
				s = strings.Replace(s, fmt.Sprintf(`[CQ:image,file=%s]`, v[1]), "", -1)
			}
			{
				t := []string{}
				for _, v := range strings.Split(s, "\n") {
					if v != "" {
						t = append(t, v)
					}
				}
				s = strings.Join(t, "\n")
			}
			if len(paths) > 0 {
				is := []tb.InputMedia{}
				for index, path := range paths {
					data, err := os.ReadFile(path)
					if err == nil {
						url := regexp.MustCompile("(https.*)").FindString(string(data))
						if url != "" {
							rsp, err := httplib.Get(url).Response()
							if err == nil {
								i := &tb.Photo{File: tb.FromReader(rsp.Body)}
								if index == 0 {
									i.Caption = s
								}
								is = append(is, i)
							}
						}
					}
				}
				b.SendAlbum(ct, is)
				return
			}
			b.Send(ct, s)
		}
		b.Handle(tb.OnPhoto, func(m *tb.Message) {
			m.Text = fmt.Sprintf(`[CQ:image,url=%s]`, m.Photo.FileURL) + m.Caption
			core.NotifyMasters(fmt.Sprintf(`[CQ:image,url=%s]`, m.Photo.FileURL) + m.Caption)
			Handler(m)
		})
		b.Handle(tb.OnText, Handler)
		logs.Info("监听telegram机器人")
		b.Start()
	}()
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
	name := sender.Message.Sender.Username
	if name == "" {
		name = fmt.Sprint(sender.Message.Sender.ID)
	}
	return name
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

func (sender *Sender) Reply(msgs ...interface{}) (int, error) {
	msg := msgs[0]
	var edit *core.Edit
	var replace *core.Replace
	for _, item := range msgs {
		switch item.(type) {
		case core.Edit:
			v := item.(core.Edit)
			edit = &v
		case core.Replace:
			v := item.(core.Replace)
			replace = &v
		case time.Duration:
			du := item.(time.Duration)
			sender.Duration = &du
		}
	}
	var rt *tb.Message
	var r tb.Recipient
	var options = []interface{}{}
	if !sender.Message.FromGroup() {
		r = sender.Message.Sender
	} else {
		r = sender.Message.Chat
		if !sender.deleted {
			options = []interface{}{&tb.SendOptions{ReplyTo: sender.Message}}
		}
	}
	var err error
	switch msg.(type) {
	case error:
		rt, err = b.Send(r, fmt.Sprintf("%v", msg), options...)
	case []byte:
		rt, err = b.Send(r, string(msg.([]byte)), options...)
	case string:
		if edit != nil && sender.reply != nil {
			if *edit == 0 {
				if sender.reply != nil {
					b.Edit(sender.reply, msg.(string))
				}
			} else {
				b.Edit(&tb.Message{
					ID: int(*edit),
				}, msg.(string))
			}
			if sender.reply == nil {
				return 0, nil
			}
			return sender.reply.ID, nil
		}
		if replace != nil {
			b.Delete(&tb.Message{
				ID: int(*edit),
			})
		}
		rt, err = b.Send(r, msg.(string), options...)
	case core.ImagePath:
		f, err := os.Open(string(msg.(core.ImagePath)))
		if err != nil {
			sender.Reply(err)
			return 0, nil
		} else {
			rts, err := b.SendAlbum(r, tb.Album{&tb.Photo{File: tb.FromReader(f)}}, options...)
			if err == nil {
				rt = &rts[0]
			}
		}
	case core.ImageUrl:
		rsp, err := httplib.Get(string(msg.(core.ImageUrl))).Response()
		if err != nil {
			sender.Reply(err)
			return 0, nil
		} else {
			rts, err := b.SendAlbum(r, tb.Album{&tb.Photo{File: tb.FromReader(rsp.Body)}}, options...)
			if err == nil {
				rt = &rts[0]
			}
		}
	}
	if err != nil {
		sender.Reply(err)
	}
	if rt != nil && sender.Duration != nil {
		if *sender.Duration != 0 {
			go func() {
				time.Sleep(*sender.Duration)
				sender.Delete()
				b.Delete(rt)
			}()
		} else {
			sender.Delete()
			b.Delete(rt)
		}
	}
	if rt != nil && sender.reply == nil {
		sender.reply = rt
	}
	if sender.reply != nil {
		return sender.reply.ID, err
	}
	return 0, nil
}

func (sender *Sender) Delete() error {
	if sender.deleted {
		return nil
	}
	msg := *sender.Message
	sender.deleted = true
	return b.Delete(&msg)
}

func (sender *Sender) Disappear(lifetime ...time.Duration) {
	if len(lifetime) == 0 {
		sender.Duration = &core.Duration
	} else {
		sender.Duration = &lifetime[0]
	}
}

func (sender *Sender) Finish() {

}

func (sender *Sender) Continue() {
	sender.goon = true
}

func (sender *Sender) IsContinue() bool {
	return sender.goon
}
