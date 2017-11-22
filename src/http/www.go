package http

import (
	"encoding/base64"
	"encoding/json"
	"g"
	"html/template"
	"io"
	"log"
	"model"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
			addr := "https://open.weixin.qq.com/connect/oauth2/authorize?appid=" + appid + "&redirect_uri=" + url.QueryEscape(fullurl) + "&response_type=code&scope=snsapi_userinfo&state=1#wechat_redirect"
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
		getuser(w, r)
		var f string // 模板文件路径
		f = filepath.Join(g.Root, "/public", "index.html")
		if !file.IsExist(f) {
			log.Println("not find", f)
			http.NotFound(w, r)
			return
		}
		data := struct {
			//Couriers 	string
			Name string
		}{
			Name: "扫描小票",
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
		file, _, _ := r.FormFile("img")
		defer file.Close()
		rate := r.FormValue("rate")
		log.Println(rate)
		rateInt, _ := strconv.Atoi(rate)
		var result model.CommonResult
		if rateInt >= 1 {
			//人工处理模块
			log.Println("save handle img:" + uuid)
			f, _ := os.Create("public/upload/" + uuid + ".jpg")
			defer f.Close()
			io.Copy(f, file)
			model.CreatNewUploadImg(uuid, openid)
			result.ErrMsg = "1" //表示有错误
			RenderJson(w, result)
			return
		}
		if file == nil || openid == "" {
			log.Println("未检测到文件")
			return
		}
		sourcebuffer := make([]byte, 4*1024*1024) //最大4M
		n, _ := file.Read(sourcebuffer)
		base64Str := base64.StdEncoding.EncodeToString(sourcebuffer[:n])
		res := model.LocalImageRecognition(base64Str)
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
			uuid := r.FormValue("uuid")
			if uuid == "" {
				return
			}
			model.DeleteUploadImg(uuid)
			http.Redirect(w, r, "/hand_operation", 302)
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
}
