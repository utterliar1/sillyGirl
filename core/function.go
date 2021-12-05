package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gin-gonic/gin"
	cron "github.com/robfig/cron/v3"
)

var c *cron.Cron

func init() {
	c = cron.New()
	c.Start()
}

type Function struct {
	Rules    []string
	FindAll  bool
	Admin    bool
	Handle   func(s Sender) interface{}
	Cron     string
	Priority int
	Disable  bool
	Server   string
}

var getPname = func() string {
	if runtime.GOOS == "windows" {
		return regexp.MustCompile(`([\w\.-]*)\.exe$`).FindStringSubmatch(os.Args[0])[0]
	}
	return regexp.MustCompile(`/([^/\s]+)$`).FindStringSubmatch(os.Args[0])[1]
}

var pname = getPname()
var name = func() string {
	return sillyGirl.Get("name", "傻妞")
}

var functions = []Function{}

var Senders chan Sender

func initToHandleMessage() {
	Senders = make(chan Sender)
	go func() {
		for {
			go handleMessage(<-Senders)
		}
	}()
}

func AddCommand(prefix string, cmds []Function) {
	for j := range cmds {
		if cmds[j].Disable {
			continue
		}
		for i := range cmds[j].Rules {
			if strings.Contains(cmds[j].Rules[i], "raw ") {
				cmds[j].Rules[i] = strings.Replace(cmds[j].Rules[i], "raw ", "", -1)
				continue
			}
			cmds[j].Rules[i] = strings.ReplaceAll(cmds[j].Rules[i], `\r\a\w`, "raw")
			if strings.Contains(cmds[j].Rules[i], "$") {
				continue
			}
			if prefix != "" {
				cmds[j].Rules[i] = prefix + `\s+` + cmds[j].Rules[i]
			}
			cmds[j].Rules[i] = strings.Replace(cmds[j].Rules[i], "(", `[(]`, -1)
			cmds[j].Rules[i] = strings.Replace(cmds[j].Rules[i], ")", `[)]`, -1)
			cmds[j].Rules[i] = regexp.MustCompile(`\?$`).ReplaceAllString(cmds[j].Rules[i], `([\s\S]+)`)
			cmds[j].Rules[i] = strings.Replace(cmds[j].Rules[i], " ", `\s+`, -1)
			cmds[j].Rules[i] = strings.Replace(cmds[j].Rules[i], "?", `(\S+)`, -1)
			cmds[j].Rules[i] = "^" + cmds[j].Rules[i] + "$"
		}
		{
			lf := len(functions)
			for i := range functions {
				f := lf - i - 1
				// logs.Warn(`functions[f].Priority %d > cmds[j].Priority %d = %t`, functions[f].Priority, cmds[j].Priority, functions[f].Priority > cmds[j].Priority)
				if functions[f].Priority > cmds[j].Priority {
					functions = append(functions[:f+1], append([]Function{cmds[j]}, functions[f+1:]...)...)
					// logs.Warn(`functions = append(functions[:f+1], append([]Function{cmds[j]}, functions[f+1:]...)...)`)
					break
				}
			}
			if len(functions) == lf {
				if lf > 0 {
					if functions[0].Priority < cmds[j].Priority && functions[lf-1].Priority < cmds[j].Priority {
						functions = append([]Function{cmds[j]}, functions...)
					} else {
						functions = append(functions, cmds[j])
					}
				} else {
					functions = append(functions, cmds[j])
				}
			}
		}

		if cmds[j].Cron != "" {
			cmd := cmds[j]
			if _, err := c.AddFunc(cmds[j].Cron, func() {
				cmd.Handle(&Faker{})
			}); err != nil {
				// logs.Warn("任务%v添加失败%v", cmds[j].Rules[0], err)
			} else {
				// logs.Warn("任务%v添加成功", cmds[j].Rules[0])
			}
		}

		if cmds[j].Server != "" {
			cmd := cmds[j]
			ss := strings.Split(cmds[j].Server, " ")
			if len(ss) > 0 {
				method := ss[0]
				path := ss[1]
				switch method {
				case "GET":
					Server.GET(path, func(c *gin.Context) {
						data, _ := ioutil.ReadAll(c.Request.Body)
						result := cmd.Handle(&Faker{
							Message: string(data),
						})
						response := make(map[string]string)
						err := json.Unmarshal([]byte(result.(string)), &response)
						if err != nil {
							c.JSON(500, map[string]string{"code": "500", "msg": "internal error"})
							return
						}
						code, ok := response["code"]
						if ok {
							c.JSON(Int(code), response)
						} else {
							c.JSON(200, map[string]string{"code": "500", "msg": "internal error"})
						}
					})
				case "POST":
					Server.POST(path, func(c *gin.Context) {
						data, _ := ioutil.ReadAll(c.Request.Body)
						result := cmd.Handle(&Faker{
							Message: string(data),
						})
						response := make(map[string]interface{})
						err := json.Unmarshal([]byte(result.(string)), &response)
						if err != nil {
							c.JSON(500, map[string]string{"code": "500", "msg": "internal error"})
							return
						}
						code, ok := response["code"]
						if ok {
							c.JSON(Int(code), response)
						} else {
							c.JSON(200, map[string]string{"code": "500", "msg": "internal error"})
						}
					})
				}
			}
		}
	}
}

func handleMessage(sender Sender) {
	content := TrimHiddenCharacter(sender.GetContent())
	defer sender.Finish()

	defer func() {
		logs.Info("%v ==> %v", sender.GetContent(), "finished")
	}()
	u, g, i := fmt.Sprint(sender.GetUserID()), fmt.Sprint(sender.GetChatID()), fmt.Sprint(sender.GetImType())
	con := true
	mtd := false
	waits.Range(func(k, v interface{}) bool {
		c := v.(*Carry)
		vs, _ := url.ParseQuery(k.(string))
		userID := vs.Get("u")
		chatID := vs.Get("c")
		imType := vs.Get("i")
		forGroup := vs.Get("f")
		if imType != i {
			return true
		}
		if chatID != g {
			return true
		}
		if userID != u && forGroup == "" {
			return true
		}
		if m := regexp.MustCompile(c.Pattern).FindString(content); m != "" {
			mtd = true
			c.Chan <- sender
			sender.Reply(<-c.Result)
			if !sender.IsContinue() {
				con = false
				return false
			}
			content = TrimHiddenCharacter(sender.GetContent())
		}
		return true
	})
	if mtd && !con {
		return
	}

	reply.Foreach(func(k, v []byte) error {
		if string(v) == "" {
			return nil
		}
		reg, err := regexp.Compile(string(k))
		if err == nil {
			if reg.FindString(content) != "" {
				sender.Reply(string(v))
			}
		}
		return nil
	})

	for _, function := range functions {
		for _, rule := range function.Rules {
			var matched bool

			if function.FindAll {
				if res := regexp.MustCompile(rule).FindAllStringSubmatch(content, -1); len(res) > 0 {
					tmp := [][]string{}
					for i := range res {
						tmp = append(tmp, res[i][1:])
					}
					sender.SetAllMatch(tmp)
					matched = true
				}
			} else {
				if res := regexp.MustCompile(rule).FindStringSubmatch(content); len(res) > 0 {
					sender.SetMatch(res[1:])
					matched = true
				}
			}
			if matched {
				logs.Info("%v ==> %v", content, rule)
				if function.Admin && !sender.IsAdmin() {
					sender.Delete()
					sender.Disappear()
					// if sender.GetImType() != "wx" && sender.GetImType() != "qq" {
					sender.Reply("再捣乱我就报警啦～")
					// }
					return
				}
				rt := function.Handle(sender)
				if rt != nil {
					sender.Reply(rt)
				}
				if sender.IsContinue() {
					sender.ClearContinue()
					content = TrimHiddenCharacter(sender.GetContent())
					goto goon
				}
				return
			}
		}
	goon:
	}

	recall := sillyGirl.Get("recall")
	if recall != "" {
		recalled := false
		for _, v := range strings.Split(recall, "&") {
			reg, err := regexp.Compile(v)
			if err == nil {
				if reg.FindString(content) != "" {
					if !sender.IsAdmin() && sender.GetImType() != "wx" {
						sender.Delete()
						sender.Reply("本妞清除了不好的消息～", time.Duration(time.Second))
						recalled = true
						break
					}
				}
			}
		}
		if recalled == true {
			return
		}
	}
}

func FetchCookieValue(ps ...string) string {
	var key, cookies string
	if len(ps) == 2 {
		if len(ps[0]) > len(ps[1]) {
			key, cookies = ps[1], ps[0]
		} else {
			key, cookies = ps[0], ps[1]
		}
	}
	match := regexp.MustCompile(key + `=([^;]*);{0,1}`).FindStringSubmatch(cookies)
	if len(match) == 2 {
		return strings.Trim(match[1], " ")
	} else {
		return ""
	}
}
