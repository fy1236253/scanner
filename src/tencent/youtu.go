package tencent

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// ExpiredInterval 鉴权过期时间
	ExpiredInterval = 86400 //一天有效时间
)

//AppSign 应用签名鉴权
type AppSign struct {
	appID     string //接入优图服务时,生成的唯一id, 用于唯一标识接入业务
	secretID  string //标识api鉴权调用者的密钥身份
	secretKey string //用于加密签名字符串和服务器端验证签名字符串的密钥，secret_key 必须严格保管避免泄露
	userID    string //接入业务自行定义的用户id，用于唯一标识一个用户, 登陆开发者账号的QQ号码
}

func (as *AppSign) orignalSign() string {
	now := time.Now().Unix()
	nonce := strconv.Itoa(rand.Intn(999999999))
	sign := fmt.Sprintf("a=%s&k=%s&e=%d&t=%d&r=%s&u=%s&f=",
		as.appID,
		as.secretID,
		now+ExpiredInterval,
		now,
		nonce,
		as.userID)
	log.Println(sign)
	return sign
}

// YTSign  优图api鉴权
func (as *AppSign) YTSign() string {
	origSign := as.orignalSign()
	h := hmac.New(sha1.New, []byte(as.secretKey))
	h.Write([]byte(origSign))
	hm := h.Sum(nil)
	//attach orig_sign to hm
	dstSign := []byte(string(hm) + origSign)
	b64 := base64.StdEncoding.EncodeToString(dstSign)
	log.Println(b64)
	return b64
}

//NewAppSign 新建应用签名
func NewAppSign(appID, secretID, secretKey, userID string) (as AppSign, err error) {
	as = AppSign{
		appID:     appID,
		secretID:  secretID,
		secretKey: secretKey,
		userID:    userID,
	}
	return
}

// YTRequest 请求参数
type YTRequest struct {
	URL   string `json:"url"`
	Appid string `json:"app_id"`
	Image string `json:"image"`
}

// YoutuRequest 发起请求
func YoutuRequest() (res string) {
	appID := "10109960"
	secretID := "AKIDAB3mKhQlU1LTZRGScaJm25XlpfKbfgPu"
	secretKey := "TihRyKn6C3f3JL8g04zgCUaR4RZZM4Zq"
	userID := "573239309"
	url := "https://api.youtu.qq.com/youtu/ocrapi/generalocr"
	as, _ := NewAppSign(appID, secretID, secretKey, userID)
	sign := as.YTSign()
	b := YTRequest{}
	b.Appid = appID
	b.Image = ImageBase64()
	data, _ := json.Marshal(b)
	resp, _ := YouTuGet(sign, url, string(data))
	log.Println(string(resp))
	return string(resp)
}

// ImageBase64 图片base64编码
func ImageBase64() string {
	f, _ := ioutil.ReadFile("photo/WechatIMG153.png")
	base64Str := base64.StdEncoding.EncodeToString(f)
	//	log.Println(base64Str)
	return base64Str
}

// YouTuGet 建立符合要求的http请求
func YouTuGet(auth, url, req string) (rsp []byte, err error) {
	//	return
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	httpreq, err := http.NewRequest("POST", url, strings.NewReader(req))
	if err != nil {
		return
	}
	httpreq.Header.Add("Authorization", auth)
	httpreq.Header.Add("Host", "api.youtu.qq.com")
	httpreq.Header.Add("Content-Type", "text/json")
	resp, err := client.Do(httpreq)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	rsp, err = ioutil.ReadAll(resp.Body)
	return
}
