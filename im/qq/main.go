package qq

import (
	"crypto/md5"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/config"
	"github.com/cdle/sillyGirl/core"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var (
	conf         *config.Config
	PasswordHash [16]byte
	AccountToken []byte
	allowStatus  = [...]client.UserOnlineStatus{
		client.StatusOnline, client.StatusAway, client.StatusInvisible, client.StatusBusy,
		client.StatusListening, client.StatusConstellation, client.StatusWeather, client.StatusMeetSpring,
		client.StatusTimi, client.StatusEatChicken, client.StatusLoving, client.StatusWangWang, client.StatusCookedRice,
		client.StatusStudy, client.StatusStayUp, client.StatusPlayBall, client.StatusSignal, client.StatusStudyOnline,
		client.StatusGaming, client.StatusVacationing, client.StatusWatchingTV, client.StatusFitness,
	}
)

var bot *coolq.CQBot
var qq = core.NewBucket("qq")

func init() {
	conf = &config.Config{}
	conf.Account.Uin = int64(qq.GetInt("uin", 0))
	conf.Account.Password = qq.Get("password")
	conf.Message.ReportSelfMessage = true
	if conf.Output.Debug {
		log.SetReportCaller(true)
	}
	logFormatter := &easy.Formatter{
		TimestampFormat: "2006/01/02 15:04:05.000",
		LogFormat:       "%time% [Q] %msg% \n",
	}
	rotateOptions := []rotatelogs.Option{
		rotatelogs.WithRotationTime(time.Hour * 24),
	}

	if conf.Output.LogAging > 0 {
		rotateOptions = append(rotateOptions, rotatelogs.WithMaxAge(time.Hour*24*time.Duration(conf.Output.LogAging)))
	}
	if conf.Output.LogForceNew {
		rotateOptions = append(rotateOptions, rotatelogs.ForceNewFile())
	}

	w, err := rotatelogs.New(path.Join("logs/qq", "%Y-%m-%d.log"), rotateOptions...)
	if err != nil {
		log.Errorf("rotatelogs init err: %v", err)
		panic(err)
	}

	log.AddHook(global.NewLocalHook(w, logFormatter, global.GetLogLevel(conf.Output.LogLevel)...))

	if device := qq.Get("device.json"); device == "" {
		client.GenRandomDevice()
		qq.Set("device.json", string(client.SystemDeviceInfo.ToJson()))
	} else {
		if err := client.SystemDeviceInfo.ReadJson([]byte(device)); err != nil {
			log.Fatalf("加载设备信息失败: %v", err)
		}
	}
	PasswordHash = md5.Sum([]byte(conf.Account.Password))
	log.Info("开始尝试登录并同步消息...")
	log.Infof("使用协议: %v", func() string {
		switch client.SystemDeviceInfo.Protocol {
		case client.IPad:
			return "iPad"
		case client.AndroidPhone:
			return "Android Phone"
		case client.AndroidWatch:
			return "Android Watch"
		case client.MacOS:
			return "MacOS"
		case client.QiDian:
			return "企点"
		}
		return "未知"
	}())
	cli = client.NewClientEmpty()
	global.Proxy = conf.Message.ProxyRewrite
	isQRCodeLogin := (conf.Account.Uin == 0 || len(conf.Account.Password) == 0) && !conf.Account.Encrypt
	isTokenLogin := false
	saveToken := func() {
		AccountToken = cli.GenToken()
		qq.Set("session.token", string(AccountToken))
	}
	if token := qq.Get("session.token"); token != "" {
		if err == nil {
			if conf.Account.Uin != 0 {
				r := binary.NewReader([]byte(token))
				cu := r.ReadInt64()
				if cu != conf.Account.Uin {
					log.Warnf("警告: 配置文件内的QQ号 (%v) 与缓存内的QQ号 (%v) 不相同", conf.Account.Uin, cu)
				}
			}
			if err = cli.TokenLogin([]byte(token)); err != nil {
				qq.Set("session.token", "")
				log.Warnf("恢复会话失败: %v , 尝试使用正常流程登录.", err)
				time.Sleep(time.Second)
				cli.Disconnect()
				cli.Release()
				cli = client.NewClientEmpty()
			} else {
				isTokenLogin = true
			}
		}
	}
	if conf.Account.Uin != 0 && PasswordHash != [16]byte{} {
		cli.Uin = conf.Account.Uin
		cli.PasswordMd5 = PasswordHash
	}
	if !isTokenLogin {
		if !isQRCodeLogin {
			if err := commonLogin(); err != nil {
				log.Fatalf("登录时发生致命错误: %v", err)
			}
		} else {
			if err := qrcodeLogin(); err != nil {
				log.Fatalf("登录时发生致命错误: %v", err)
			}
		}
	}
	var times uint = 10 // 重试次数
	var reLoginLock sync.Mutex
	cli.OnDisconnected(func(_ *client.QQClient, e *client.ClientDisconnectedEvent) {
		reLoginLock.Lock()
		defer reLoginLock.Unlock()
		times = 1
		if cli.Online {
			return
		}
		log.Warnf("Bot已离线: %v", e.Message)
		time.Sleep(time.Second * time.Duration(conf.Account.ReLogin.Delay))
		for {
			if conf.Account.ReLogin.Disabled {
				os.Exit(1)
			}
			if times > conf.Account.ReLogin.MaxTimes && conf.Account.ReLogin.MaxTimes != 0 {
				log.Fatalf("Bot重连次数超过限制, 停止")
			}
			times++
			if conf.Account.ReLogin.Interval > 0 {
				log.Warnf("将在 %v 秒后尝试重连. 重连次数：%v/%v", conf.Account.ReLogin.Interval, times, conf.Account.ReLogin.MaxTimes)
				time.Sleep(time.Second * time.Duration(conf.Account.ReLogin.Interval))
			} else {
				time.Sleep(time.Second)
			}
			log.Warnf("尝试重连...")
			err := cli.TokenLogin(AccountToken)
			if err == nil {
				saveToken()
				return
			}
			log.Warnf("快速重连失败: %v", err)
			if isQRCodeLogin {
				log.Fatalf("快速重连失败, 扫码登录无法恢复会话.")
			}
			log.Warnf("快速重连失败, 尝试普通登录. 这可能是因为其他端强行T下线导致的.")
			time.Sleep(time.Second)
			if err := commonLogin(); err != nil {
				log.Errorf("登录时发生致命错误: %v", err)
			} else {
				saveToken()
				break
			}
		}
	})
	saveToken()
	cli.AllowSlider = true
	log.Infof("登录成功 欢迎使用: %v", cli.Nickname)
	global.Check(cli.ReloadFriendList(), true)
	global.Check(cli.ReloadGroupList(), true)
	if conf.Account.Status >= int32(len(allowStatus)) || conf.Account.Status < 0 {
		conf.Account.Status = 0
	}
	cli.SetOnlineStatus(allowStatus[int(conf.Account.Status)])
	bot = coolq.NewQQBot(cli, conf)
	_ = bot.Client
	coolq.SetMessageFormat("string")
	onPrivateMessage := func(c *client.QQClient, m *message.PrivateMessage) {
		core.Senders <- &Sender{
			Message: m,
		}

		// cqm := coolq.ToStringMessage(m.Elements, 0, true)
		// fmt.Println(cqm, m.Self, m.Target, m.Sender.Uin)
		if m.Sender.Uin != c.Uin {
			c.MarkPrivateMessageReaded(m.Sender.Uin, int64(m.Time))
		}
	}
	OnGroupMessage := func(c *client.QQClient, m *message.GroupMessage) {
		// cqm := coolq.ToStringMessage(m.Elements, m.GroupCode, true)
		// fmt.Println(cqm, m.GroupCode, m.Sender.Uin)
		core.Senders <- &Sender{
			Message: m,
		}
	}
	bot.Client.OnPrivateMessage(onPrivateMessage)
	bot.Client.OnSelfPrivateMessage(onPrivateMessage)
	bot.Client.OnGroupMessage(OnGroupMessage)
	bot.Client.OnSelfGroupMessage(OnGroupMessage)
}
