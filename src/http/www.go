package http

import (
	"encoding/base64"
	"encoding/json"
	"g"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"model"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"tencent"
	"time"
	"util"

	"github.com/toolkits/file"
)

func getuser(w http.ResponseWriter, r *http.Request) {
	fullurl := "http://" + r.Host + r.RequestURI
	appid := g.Config().Wechats[0].AppID
	appsecret := g.Config().Wechats[0].AppSecret

	// 参数检查
	queryValues, err := url.ParseQuery(r.URL.RawQuery)
	log.Println("ParseQuery", queryValues)
	if err != nil {
		log.Println("[ERROR] URL.RawQuery", err)
		w.WriteHeader(400)
		return
	}

	// 从 session 中获取用户的 openid
	sess, _ := globalSessions.SessionStart(w, r)
	defer sess.SessionRelease(w)
	if sess.Get("openid") == nil {
		sess.Set("openid", "")
	}
	openid := sess.Get("openid").(string)
	log.Println(openid)
	// session 不存在
	if openid == "" {
		//oauth 跳转 ， 页面授权获取用户基本信息
		code := queryValues.Get("code") //  摇一摇入口 code 有效
		state := queryValues.Get("state")
		if code == "" && state == "" {
			addr := "https://open.weixin.qq.com/connect/oauth2/authorize?appid=" + appid + "&redirect_uri=" + url.QueryEscape(fullurl) + "&response_type=code&scope=snsapi_base&state=1#wechat_redirect"
			log.Println("http.Redirect", addr)
			http.Redirect(w, r, addr, 302)
			return
		}
		// 获取用户信息
		openid, _ = util.GetAccessTokenFromCode(appid, appsecret, code)
		if openid == "" {
			return
		}
		sess.Set("openid", openid)
	}
	return
}

// ConfigWebHTTP 对外http
func ConfigWebHTTP() {

	// 用户上传图片
	http.HandleFunc("/scanner", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fullurl := "http://" + r.Host + r.RequestURI
		wxid := g.Config().Wechats[0].WxID
		appid := g.Config().Wechats[0].AppID
		nonce := strconv.Itoa(rand.Intn(999999999))
		ts := time.Now().Unix()
		sign := util.WXConfigSign(g.GetJsAPITicket(), nonce, strconv.FormatInt(ts, 10), fullurl)
		sess, _ := globalSessions.SessionStart(w, r)
		defer sess.SessionRelease(w)
		user := r.FormValue("openid")
		log.Println(user)
		if sess.Get("openid") == nil {
			sess.Set("openid", user)
		}
		var f string // 模板文件路径
		f = filepath.Join(g.Root, "/public", "index.html")
		if !file.IsExist(f) {
			log.Println("not find", f)
			http.NotFound(w, r)
			return
		}
		data := struct {
			WxID  string
			AppID string
			Nonce string
			Sign  string
		}{
			WxID:  wxid,
			AppID: appid,
			Nonce: nonce,
			Sign:  sign,
		}
		t, err := template.ParseFiles(f)
		err = t.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
		return
	})

	// 上传图片后  返回识别结果
	http.HandleFunc("/consumer", func(w http.ResponseWriter, r *http.Request) {
		var f string // 模板文件路径
		queryValues, _ := url.ParseQuery(r.URL.RawQuery)
		uuid := queryValues.Get("uuid")
		f = filepath.Join(g.Root, "/public", "scanFinish.html")
		if !file.IsExist(f) {
			log.Println("not find", f)
			http.NotFound(w, r)
			return
		}
		if uuid == "" {
			log.Println("[error]:have no uuid")
			return
		}
		// 基本参数设置
		log.Println(uuid)
		info := model.QueryImgRecord(uuid)
		data := struct {
			UUID string
			Info *model.IntegralReq
		}{
			UUID: uuid,
			Info: info,
		}
		log.Println(info)
		t, err := template.ParseFiles(f)
		// log.Println(err)
		err = t.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
		return
	})
	http.HandleFunc("/credits", func(w http.ResponseWriter, r *http.Request) {

		queryValues, _ := url.ParseQuery(r.URL.RawQuery)
		var f string // 模板文件路径
		f = filepath.Join(g.Root, "/public", "scannerIndex.html")
		if !file.IsExist(f) {
			log.Println("not find", f)
			http.NotFound(w, r)
			return
		}
		score := queryValues.Get("score")
		// 基本参数设置
		data := struct {
			Score string
		}{
			Score: score,
		}
		t, err := template.ParseFiles(f)
		err = t.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
		return
	})
	http.HandleFunc("/uploadImg", func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		r.ParseMultipartForm(32 << 20)
		sess, _ := globalSessions.SessionStart(w, r)
		defer sess.SessionRelease(w)
		if sess.Get("openid") == nil {
			log.Println("需要在微信中打开")
		}
		openid := sess.Get("openid").(string)
		uuid := model.CreateNewID(12)
		f, _, _ := r.FormFile("img")
		defer f.Close()
		rate := r.FormValue("rate")
		log.Println(rate)
		rateInt, _ := strconv.Atoi(rate)
		var result model.CommonResult
		if rateInt > 1 {
			//人工处理模块
			log.Println("save handle img:" + uuid)
			fs, _ := os.Create("public/upload/" + uuid + ".jpg")
			defer f.Close()
			io.Copy(fs, f)
			model.CreatNewUploadImg(uuid, openid)
			result.ErrMsg = "1" //表示有错误
			RenderJson(w, result)
			return
		}
		if f == nil || openid == "" {
			log.Println("未检测到文件")
			return
		}
		imgByte, _ := ioutil.ReadAll(f)
		base64Str := base64.StdEncoding.EncodeToString(imgByte)
		var res *tencent.IntegralReq
		recongnition, types := tencent.YoutuRequest(base64Str)
		log.Println(types)
		if types == 2 {
			res = tencent.TicketHandle(recongnition)
		} else {
			res = tencent.TicketHandleSecond(recongnition)
		}
		result.ErrMsg = "success"
		if res == nil {
			//识别有错误  返回错误
			log.Println("fail to upload")
			result.ErrMsg = "1"
			RenderJson(w, result)
			return
		} else {
			result.DataInfo = res
			result.UUID = uuid
		}
		log.Println(uuid)
		drugInfo, _ := json.Marshal(res)
		model.CreatImgRecord(uuid, openid, string(drugInfo)) //上传记录上传至数据库记录
		RenderJson(w, result)
		log.Println(time.Since(t))
		return
	})
	http.HandleFunc("/hand_operation", func(w http.ResponseWriter, r *http.Request) {
		imgItems := model.GetUploadImgInfo()
		var f string // 模板文件路径
		f = filepath.Join(g.Root, "/public", "handOperation.html")
		if !file.IsExist(f) {
			log.Println("not find", f)
			http.NotFound(w, r)
			return
		}
		// 基本参数设置
		data := struct {
			//Couriers 	string
			ImgItems []string
		}{
			ImgItems: imgItems,
		}

		t, err := template.ParseFiles(f)
		err = t.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
		return
	})
	http.HandleFunc("/save_jifen_info", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		uuid := r.FormValue("uuid")
		sess, _ := globalSessions.SessionStart(w, r)
		defer sess.SessionRelease(w)
		openid := sess.Get("openid").(string)
		if openid == "" {
			log.Println("用户登录失败")
			return
		}
		pkg := model.QueryImgRecord(uuid)
		pkg.Openid = openid
		pkg.Times = time.Now().Unix()
		drug := new(model.MedicineList)
		pkg.Medicine = append(pkg.Medicine, drug)
		result := model.GetIntegral(pkg)
		RenderJson(w, result)
		return
	})
	http.HandleFunc("/edit_img", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Method == "POST" {
			log.Println(r.Form)
			var p model.IntegralReq
			var m model.MedicineList
			uuid := r.FormValue("uuid")
			openid := model.GetOpenidByUID(uuid)
			amount := r.FormValue("amount")
			log.Println(openid)
			drugArr := r.Form["drug[]"]    //药品名称
			drugCount := r.Form["count[]"] //药品数量
			drugPrice := r.Form["price[]"] //药品价格
			p.Openid = openid
			p.Shop = r.FormValue("name")
			p.OrderId = r.FormValue("order")
			p.TotalFee, _ = strconv.ParseFloat(amount, 64)
			p.Times = time.Now().Unix()
			for k, v := range drugArr {
				m.Name = v
				m.Amount, _ = strconv.Atoi(drugCount[k])
				m.Money, _ = strconv.ParseFloat(drugPrice[k], 64)
				p.Medicine = append(p.Medicine, &m)
			}
			log.Println(p)
			result := model.GetIntegral(&p)
			if result.Success == "t" {
				model.DeleteUploadImg(uuid)
			}
			RenderJson(w, result)
			return
		}
		urlParse, _ := url.ParseQuery(r.URL.RawQuery)
		uuid := urlParse.Get("uuid")
		// log.Println(uuid)
		var f string // 模板文件路径
		f = filepath.Join(g.Root, "/public", "edit.html")
		if !file.IsExist(f) {
			log.Println("not find", f)
			http.NotFound(w, r)
			return
		}
		// 基本参数设置
		data := struct {
			UUID string
		}{
			UUID: uuid,
		}

		t, err := template.ParseFiles(f)
		err = t.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
		return
	})
	http.HandleFunc("/handle", func(w http.ResponseWriter, r *http.Request) {
		model.ImportDatbase()
		return
	})
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		RenderJson(w, `{"test":"test"}`)
		return
	})
}
