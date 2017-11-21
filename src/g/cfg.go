package g

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/toolkits/file"
)

var (
	//ConfigFile 配置文件
	ConfigFile string
	//WXconfig 微信解析后的配置信息
	WXconfig   *GlobalConfig
	configLock = new(sync.RWMutex)
)

//GlobalConfig 微信配置结构体
type GlobalConfig struct {
	Logs    string          `json:"logs"`
	HTTP    *HTTPConfig     `json:"http"`
	DB      *DBConfig       `json:"db"`
	Wechats []*WechatConfig `json:"wechats"`
}

//HTTPConfig 端口绑定
type HTTPConfig struct {
	Enable bool   `json:"enable"`
	Listen string `json:"listen"`
}

//DBConfig mysql配置
type DBConfig struct {
	Dsn     string `json:"dsn"`
	MaxIdle int    `json:"maxIdle"`
}

//WechatConfig 微信公众号配置，支持多个
type WechatConfig struct {
	WxID      string `json:"WxId"`
	AppSecret string `json:"AppSecret"`
	AppID     string `json:"AppId"`
}

//Config 获取配置信息
func Config() *GlobalConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return WXconfig
}

// ParseConfig 解析配置文件
func ParseConfig(cfg string) {
	ConfigLock := new(sync.RWMutex)
	if cfg == "" {
		log.Fatalln("config file not specified: use -c $filename")
	}
	if !file.IsExist(cfg) {
		log.Fatalln("config file specified not found:", cfg)
	}

	ConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file", cfg, "error:", err.Error())
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file", cfg, "error:", err.Error())
	}
	// set config
	ConfigLock.Lock()
	defer ConfigLock.Unlock()
	WXconfig = &c
	log.Println("g.ParseConfig ok, file", cfg)
}
