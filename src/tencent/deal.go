package tencent

import (
	"encoding/json"
	"g"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// YTResponse 优图返回内容
type YTResponse struct {
	ErrorCode int      `json:"errorcode"`
	ErrorMsg  string   `json:"errormsg"`
	Items     []YTItem `json:"items"`
}

// YTItem 优图具体信息
type YTItem struct {
	Itemcoord struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"itemcoord"`
	Itemstring string `json:"itemstring"`
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

// TicketHandle 小票处理）第一种
func TicketHandle(ytJSON string) *IntegralReq {
	var res *YTResponse
	var amountFloat, amount float64
	var unionid, shop string
	var drugName string
	var drugItem []*MedicineList
	var sortAmount sort.Float64Slice
	json.Unmarshal([]byte(ytJSON), &res)
	if len(res.Items) == 0 {
		log.Println(res)
		return nil
	}
	for _, v := range res.Items { //轮训关键字
		// 商铺名称查找
		name := recongnitionName(v.Itemstring)
		if name != "" {
			log.Println(name)
			shop = name
		}
		// 小票金额查找
		amountFloat = recongnitionAmount(v.Itemstring)
		sortAmount = append(sortAmount, amountFloat)
		// 小票单号查找
		id := RecongnitionOrderNum(v.Itemstring)
		if id != "" {
			unionid = id
		}
		// 药品查找
		drug := recongnitionDrug(v.Itemstring)
		if drug != "" {
			drugName = SelectDrugInfo(drug)
			if drugName != "" {
				nameList := new(MedicineList)
				nameList.Name = drugName
				drugItem = append(drugItem, nameList)
			}
		}
	}
	// 轮训金额 取第二大的
	sortAmount.Sort()
	if len(sortAmount) < 2 && len(sortAmount) >= 1 {
		amount = sortAmount[len(sortAmount)-1]
	} else if len(sortAmount) >= 2 {
		amount = sortAmount[len(sortAmount)-2]
	} else {
		amount = 0
	}
	// 返回结果拼接
	result := new(IntegralReq)
	result.TotalFee = amount
	// r := rand.New(rand.NewSource(time.Now().UnixNano())) + strconv.Itoa(r.Intn(100))
	result.OrderId = unionid
	result.Shop = shop
	result.Medicine = drugItem
	// 三个必须参数不能为空
	if shop == "" || unionid == "" || 0 == amount {
		log.Println("order info have error")
		return nil
	}
	log.Println(result)
	return result
}

// TicketHandleSecond 小票处理第二种
func TicketHandleSecond(ytJSON string) *IntegralReq {
	t := time.Now()
	var res *YTResponse
	var amountFloat, amount float64
	var orderID, unitName, drugName string
	var topDistance int
	var drugItem []*MedicineList
	result := new(IntegralReq)
	json.Unmarshal([]byte(ytJSON), &res)
	topDistance = SecondMidStr(res)
	for _, v := range res.Items { //轮训关键字
		order := SecondRecongnitionOrderNum(v.Itemstring)
		if order != "" {
			orderID = order
		}
		amountFloat = recongnitionAmount(v.Itemstring)
		if amountFloat > amount && v.Itemcoord.Y < topDistance {
			amount = amountFloat
		}
		name := recongnitionName(v.Itemstring)
		if name != "" {
			unitName = name
		}
		drug := recongnitionDrug(v.Itemstring)
		if drug != "" {
			//			log.Println("匹配到：" + drug)
			drugName = SelectDrugInfo(drug)
			if drugName != "" {
				log.Println(drugName)
				nameList := new(MedicineList)
				nameList.Name = drugName
				drugItem = append(drugItem, nameList)
			}
		}
	}
	result.TotalFee = amount
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))  + strconv.Itoa(r.Intn(100))
	result.OrderId = orderID
	result.Shop = unitName
	result.Medicine = drugItem
	log.Println(unitName)
	log.Println(orderID)
	log.Println(amount)
	if unitName == "" || orderID == "" || 0 == amount {
		log.Println("order info have error")
		return nil
	}
	log.Printf("our api time:%v", time.Since(t))
	return result
}

// RecongnitionOrderNum 处理订单中的编号
func RecongnitionOrderNum(str string) string { //加上单据号搜索
	regular := `^(单据号|单据).*\d+|\d{15}`
	match, name := commonMatch(regular, str)
	log.Println(name)
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
	match, amount := commonMatch(regular, str)
	if match {
		amountFloat, _ := strconv.ParseFloat(amount, 64)
		return amountFloat
	}
	return 0
}

// SelectDrugInfo 药品信息
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
	regular := `.*.(大药房)|.*.(连锁店|连锁)`
	match, name := commonMatch(regular, str)
	name = strings.Replace(name, "落款单位:", "", -1)
	if match {
		return name
	}
	return ""
}

// recongnitionDrug 识别药品
func recongnitionDrug(str string) string {
	regular := `\p{Han}{2,}`
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

// SecondMidStr 第二种小票查询中间字符位置
func SecondMidStr(result *YTResponse) int {
	regular := `.*.(费用分类)|医保费用分类`
	for _, v := range result.Items {
		match, _ := commonMatch(regular, v.Itemstring)
		if match {
			return v.Itemcoord.Y
		}
	}
	return 0
}

// SecondRecongnitionOrderNum 第二种识别
func SecondRecongnitionOrderNum(str string) string { //加上单据号搜索
	regular := `[^\d]+\d{7}$`
	match, name := commonMatch(regular, str)
	if match {
		// log.Println("单号" + name[len(name)-7:])
		return name[len(name)-7:]
	}
	return ""
}

// RecongnitionType 判断小票的类型
func RecongnitionType(str string) (types int) {
	var res YTResponse
	json.Unmarshal([]byte(str), &res)
	regular := `^(姓名|人员性质|收款人)`
	for _, v := range res.Items {
		match, _ := commonMatch(regular, v.Itemstring)
		if match {
			types = 2
		}
	}
	return types
}
