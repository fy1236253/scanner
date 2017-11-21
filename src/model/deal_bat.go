package model

import (
	"encoding/json"
	"g"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/toolkits/net/httplib"
)

// BATList 百度数据返回
type BATList struct {
	Words string `json:"words"` //目前只正则匹配汉字
}

// BATResult 识别的列表
type BATResult struct {
	WordsResult []BATList `json:"words_result"`
}

// RecognizeResult 自处理后的结果
type RecognizeResult struct {
	ShopName    string  `json:"shop_name"`
	TotalAmount float64 `json:"total_amount"`
	Unionid     string  `json:"unionid"`
}

// CommonResult api接口返回数据
type CommonResult struct {
	ErrMsg   string      `json:"errMsg"`
	DataInfo interface{} `json:"data"`
	UUID     string      `json:"uuid"`
}

// IntegralReq 积分请求
type IntegralReq struct {
	Openid   string          `json:"openid"`
	Shop     string          `json:"shop"`
	OrderId  string          `json:"order_id"`
	TotalFee float64         `json:"total_fee"`
	Times    int64           `json:"timestamp"`
	Medicine []*MedicineList `json:"medicine"`
}

// MedicineList 药品信息
type MedicineList struct {
	Name   string  `json:"name"`
	Amount int     `json:"amount"`
	Money  float64 `json:"money"`
}
type IntegralResponse struct {
	Success string `json:"success"`
	Error   string `json:"error"`
	Result  string `json:"result"`
}

// GetIntegral 积分请求
func GetIntegral(pkg *IntegralReq) (response *IntegralResponse) {
	url := "http://101.200.187.60:8180/members/servlet/ACSClientHttp"
	req := httplib.Post(url)
	req.Param("methodName", "getReceipt")
	req.Param("beanName", "appuserinfohttpservice")
	req.Param("appcode", "AIDAOKE")
	req.Param("imei", "0001")
	ps, _ := json.Marshal(pkg)
	req.Param("json", string(ps))
	log.Println(string(ps))
	resp, _ := req.String()
	log.Println(resp)
	json.Unmarshal([]byte(resp), &response)
	return response
}

// BatImageRecognition 百度的图像识别接口
func BatImageRecognition(base64Str string) string {
	url := "https://aip.baidubce.com/rest/2.0/ocr/v1/accurate?access_token=24.1f248484d5b7faf54537dfae92fed52c.2592000.1512598910.282335-10330945"
	req := httplib.Post(url).SetTimeout(3*time.Second, 1*time.Minute)
	req.Header("Content-Type", "application/x-www-form-urlencoded")
	// req.Body("{\"img\":" + base64Str + "}")
	req.Param("image", base64Str)
	resp, err := req.String()
	if err != nil {
		log.Println(err)
		return ""
	}
	return resp
}

// LocalImageRecognition 自由图片处理 提取数据
func LocalImageRecognition(base64 string) *IntegralReq {
	t := time.Now()
	resp := BatImageRecognition(base64)
	log.Printf("bat time:%v", time.Since(t))
	if resp == "" {
		log.Println("request BAT fail")
		return nil
	}
	// log.Println(resp)
	var res BATResult
	var amountFloat, amount float64
	var unionid, shop string
	result := new(IntegralReq)
	json.Unmarshal([]byte(resp), &res)
	var drugName string
	var drugItem []*MedicineList
	for _, v := range res.WordsResult { //轮训关键字
		// log.Println(v)
		name := recongnitionName(v.Words)
		if name != "" {
			shop = name
		}
		amountFloat = recongnitionAmount(v.Words)
		if amountFloat >= amount {
			amount = amountFloat
		}
		id := recongnitionOrderNum(v.Words)
		if id != "" {
			unionid = id
		}
		drug := recongnitionDrug(v.Words)
		if drug != "" {
			//			log.Println("匹配到：" + drug)
			drugName = SelectDrugInfo(drug)
			if drugName != "" {
				//				log.Println(drugName)
				nameList := new(MedicineList)
				nameList.Name = drugName
				drugItem = append(drugItem, nameList)
			}
		}
	}
	result.TotalFee = amount
	result.OrderId = unionid
	result.Shop = shop
	result.Medicine = drugItem
	if shop == "" || unionid == "" || 0 == amount {
		log.Println("order info have error")
		return nil
	}
	log.Printf("our api time:%v", time.Since(t))
	return result
}

// RecongnitionOrderNum 处理订单中的编号
func recongnitionOrderNum(str string) string { //加上单据号搜索
	regular := `^(单据号|单据).\d[0-9]+|\d{15}`
	match, name := commonMatch(regular, str)
	reg := regexp.MustCompile("[\u4E00-\u9FA5].")
	name = reg.ReplaceAllLiteralString(name, "")
	if match {
		return name
	}
	return ""
}

// recongnitionAmount 识别订单中的金额
func recongnitionAmount(str string) float64 {
	regular := `\d+\..*\d+`
	//	regular := `(-?\d*)\.?\d+`
	match, amount := commonMatch(regular, str)
	if match {
		amountFloat, _ := strconv.ParseFloat(amount, 64)
		return amountFloat
	}
	return 0
}

func SelectDrugInfo(str string) string {
	drug := g.DrugConfig()
	for _, v := range drug {
		regular := `(` + str + `)`
		match, _ := commonMatch(regular, v)
		if match {
			return v
		}
	}
	return ""
}

// recongnitionName 匹配订单中的药店名称
func recongnitionName(str string) string {
	regular := `.*.(大药房)|.*.(连锁店)`
	match, name := commonMatch(regular, str)
	name = strings.Replace(name, "落款单位:", "", -1)
	if match {
		return name
	}
	return ""
}

// recongnitionDrug 识别药品
func recongnitionDrug(str string) string {
	regular := `\p{Han}{3,}`
	match, drug := commonMatch(regular, str)
	if match {
		return drug
	}
	return ""
}

// commonMatch 通用正则匹配
func commonMatch(regular, str string) (bool, string) {
	reg := regexp.MustCompile(regular)
	name := reg.FindAllString(str, -1)
	match := reg.MatchString(str)
	if match {
		//		log.Println(name)
		return true, name[0]
	}
	return false, ""
}

// CreateNewID 根据linux系统生成
func CreateNewID(n int) string {
	out, err := exec.Command("uuidgen").Output() //C4817373-A378-412B-BC00-A619730FD9C1
	if err != nil {
		log.Fatal(err)
	}
	outStr := string(out)
	uuid := outStr[:n]
	return uuid
}
