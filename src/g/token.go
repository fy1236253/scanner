package g

import (
	"sync"
	"time"
	"util"
)

// Token  全局token
// JsTicket js
var (
	Token    string
	JsTicket string
	JsLock   = new(sync.RWMutex)
)

// Expires token的过期时间
var Expires int

// TokenRes token返回值
type TokenRes struct {
	AccessToken string `json:"access_token"`
}

// AccessTokenRequest 获取accesstoken
func AccessTokenRequest() (token string) {
	appid := Config().Wechats[0].AppID
	secret := Config().Wechats[0].AppSecret
	res := util.GetToken(appid, secret)
	token = res.Token
	return
}

// StartToken 进程中维护token
func StartToken() {
	go TokenCacheInit()
}

//TokenCacheInit 缓存token
func TokenCacheInit() {
	for {
		if Expires <= 0 {
			setToken()
			SetJsAPITicket()
			Expires = int(7000)
		} else {
			time.Sleep(1 * time.Second)
			Expires--
		}
	}
}

func setToken() {
	JsLock.Lock()
	defer JsLock.Unlock()
	Token = AccessTokenRequest()
}

// SetJsAPITicket ticket设置
func SetJsAPITicket() {
	JsLock.Lock()
	defer JsLock.Unlock()
	JsTicket = util.GetJsApiTicket(Token)
}

// GetJsAPITicket 获取ticket
func GetJsAPITicket() string {
	JsLock.Lock()
	defer JsLock.Unlock()
	return JsTicket
}
