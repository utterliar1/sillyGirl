package core

import "regexp"

var Pushs = map[string]func(int, string){}
var GroupPushs = map[string]func(int, int, string){}

func Push(class string, uid int, content string) {
	if push, ok := Pushs[class]; ok {
		push(uid, content)
	}
}

type Chat struct {
	Class  string
	ID     int
	UserID int
}

func (ct *Chat) Push(content interface{}) {
	switch content.(type) {
	case string:
		if push, ok := GroupPushs[ct.Class]; ok {
			push(ct.ID, ct.UserID, content.(string))
		}
	case error:
		if push, ok := GroupPushs[ct.Class]; ok {
			push(ct.ID, ct.UserID, content.(error).Error())
		}
	}
}

func NotifyMasters(content string) {
	if sillyGirl.GetBool("ignore_notify", false) == true {
		return
	}
	for _, class := range []string{"tg", "qq"} {
		notify := Bucket(class).Get("notifiers")
		if notify == "" {
			notify = Bucket(class).Get("masters")
		}
		for _, v := range regexp.MustCompile(`(\d+)`).FindAllStringSubmatch(notify, -1) {
			if push, ok := Pushs[class]; ok {
				push(Int(v[1]), content)
			}
		}
	}
}
