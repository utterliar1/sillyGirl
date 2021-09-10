package core

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/beego/beego/v2/adapter/httplib"
	"github.com/buger/jsonparser"
	"github.com/cdle/sillyGirl/im"
)

type Reply struct {
	Rules   []string
	Type    string //text url
	Content string
	Request struct {
		Url          string
		Method       string
		Body         string // form-data raw
		Headers      []string
		ResponseType string `yaml:"response_type"` //text json image
		Get          string
	}
}

func InitReplies() {
	for _, v := range Config.Replies {
		reply := v
		var handler func(s im.Sender) interface{}
		if reply.Type != "url" {
			handler = func(s im.Sender) interface{} {
				return reply.Content
			}
		}
		handler = func(s im.Sender) interface{} {
			url := reply.Request.Url
			body := reply.Request.Body
			for k, v := range s.GetMatch() {
				url = strings.Replace(url, fmt.Sprintf(`{{%d}}`, k+1), v, -1)
				body = strings.Replace(body, fmt.Sprintf(`{{%d}}`, k+1), v, -1)
			}
			var req httplib.BeegoHTTPRequest
			if strings.ToLower(reply.Request.Method) == "post" {
				req = *httplib.Post(url)
			} else {
				req = *httplib.Get(url)
			}
			for _, header := range reply.Request.Headers {
				ss := strings.Split(header, ":")
				if len(ss) > 0 {
					req.Header(ss[0], strings.Join(ss[1:], ":"))
				}
			}
			if reply.Request.Body != "" {
				req.Body(body)
			}
			rsp, err := req.Response()
			if err != nil {
				if reply.Content != "" {
					s.Reply(reply.Content)
				} else {
					s.Reply(err)
				}
				return nil
			}
			switch reply.Request.ResponseType {
			case "image":
				s.Reply(rsp)
			case "json":
				d, _ := ioutil.ReadAll(rsp.Body)
				f, err := jsonparser.GetString(d, strings.Split(reply.Request.Get, ".")...)
				if err != nil {
					s.Reply(err)
					return true
				}
				s.Reply(f)
			default:
				d, _ := ioutil.ReadAll(rsp.Body)
				fmt.Println(string(d))
				s.Reply(d)
			}
			return nil
		}
		functions = append(functions, Function{
			Rules:  reply.Rules,
			Handle: handler,
		})

	}
}
