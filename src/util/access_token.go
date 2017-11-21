package util

import (
	"encoding/json"
	"github.com/toolkits/net/httplib"
	"log"
	"net/url"
	"time"
)

type AccessTokenInfo struct {
	ErrCode   int64  `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	Token     string `json:"access_token"`
	ExpiresIn int64  `json:"expires_in"` // 有效时间, seconds
}

func GetToken(appId, appSecret string) *AccessTokenInfo {

	url := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + url.QueryEscape(appId) +
		"&secret=" + url.QueryEscape(appSecret)

	r := httplib.Get(url).SetTimeout(3*time.Second, 1*time.Minute)
	resp, err := r.String() //  {"errcode":40013,"errmsg":"invalid appid hint: [1HtmMa0495vr19]"}
	//log.Println(resp)

	if err != nil {
		log.Println("[ERROR] refresh token", err)
		return nil
	}

	var token AccessTokenInfo

	if err = json.Unmarshal([]byte(resp), &token); err != nil {
		log.Println("[ERROR] json ", err, resp)
		return nil
	}

	if token.ErrCode != 0 {
		//log.Println("[ERROR]", token.ErrCode, token.ErrMsg, appId)
		return nil
	}

	//log.Println(token)

	return &token
}

type JsApiTicketInfo struct {
	ErrCode   int64  `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	Ticket    string `json:"ticket"`
	ExpiresIn int64  `json:"expires_in"` // 有效时间, seconds
}

func GetJsApiTicket(access_token string) string {
	incompleteURL := "https://api.weixin.qq.com/cgi-bin/ticket/getticket?type=jsapi&access_token=" + url.QueryEscape(access_token)

	req := httplib.Get(incompleteURL).SetTimeout(3*time.Second, 1*time.Minute)

	resp, err := req.String()

	log.Println(resp)

	if err != nil {
		log.Println("[ERROR]", err)
		return ""
	}

	var result JsApiTicketInfo
	err = json.Unmarshal([]byte(resp), &result)
	if result.ErrCode != 0 {
		log.Println("[ERROR]", result)
		return ""
	}
	return result.Ticket
}

/*{
   "access_token":"ACCESS_TOKEN",
   "expires_in":7200,
   "refresh_token":"REFRESH_TOKEN",
   "openid":"OPENID",
   "scope":"SCOPE",
   "unionid": "o6_bmasdasdsad6_2sgVt7hMZOPfL"
}*/
type WebTokenInfo struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`

	Token        string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"` // 有效时间, seconds
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionId      string `json:"unionid"`
}

// 网页授权获取用户信息 接口 ， 通过 code 临时获取  token 和 openid
func GetAccessTokenFromCode(appId, appSecret, code string) (openid, token string) {
	u := "https://api.weixin.qq.com/sns/oauth2/access_token?appid=" + appId + "&secret=" + appSecret + "&code=" + code + "&grant_type=authorization_code"

	req := httplib.Get(u).SetTimeout(3*time.Second, 1*time.Minute)

	resp, err := req.String()

	//log.Println(resp)

	if err != nil {
		log.Println("[ERROR]", err)
		return "", ""
	}

	var result WebTokenInfo
	err = json.Unmarshal([]byte(resp), &result)
	if result.ErrCode != 0 {
		log.Println("[ERROR]", result)
		return "", ""
	}

	return result.OpenId, result.Token
}
