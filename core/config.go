package core

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/logs"
	"github.com/beego/beego/v2/adapter/httplib"
	"github.com/cdle/sillyGirl/im"
	"gopkg.in/yaml.v2"
)

type Yaml struct {
	Im      []im.Config
	Replies []Reply
}

var ExecPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))

var Config Yaml

func init() {
	ReadYaml(ExecPath+"/conf/", &Config, "https://raw.githubusercontent.com/cdle/sillyGirl/main/conf/demo_config.yaml")
	InitReplies()
	initToHandleMessage()
}

func ReadYaml(confDir string, conf interface{}, url string) {
	path := confDir + "config.yaml"
	if _, err := os.Stat(confDir); err != nil {
		os.MkdirAll(confDir, os.ModePerm)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		logs.Warn(err)
	}
	s, _ := ioutil.ReadAll(f)
	if len(s) == 0 {
		logs.Info("下载配置%s", url)
		r, err := httplib.Get("https://ghproxy.com/" + url).Response()
		if err == nil {
			io.Copy(f, r.Body)
		}
	}
	f.Close()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		logs.Warn("解析配置文件%s读取错误: %v", path, err)
		return
	}
	if yaml.Unmarshal(content, conf) != nil {
		logs.Warn("解析配置文件%s出错: %v", path, err)
		return
	}
	logs.Info("解析配置文件%s", path)
}
