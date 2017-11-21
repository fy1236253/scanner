package g

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/toolkits/file"
)

const (
	// VERSION 版本号
	VERSION = "wechatv1 0.1.0"
)

// Root 获取当前路径
var Root string

// InitRootDir 初始化路径
func InitRootDir() {
	var err error
	Root, err = os.Getwd()
	if err != nil {
		log.Fatalln("getwd fail:", err)
	}
}

var (
	//DrugFile 药品文件
	DrugFile []string
	DrugLock = new(sync.RWMutex)
)

// ParseDrugConfig 药品名称
func ParseDrugConfig() {
	f, _ := os.Open("data/medic.json")
	var arry []string
	r := bufio.NewReader(f)
	for {
		line, err := file.ReadLine(r)
		if err == io.EOF {
			break
		}
		str := string(line)
		str = strings.Trim(str, " ")
		if str == "" {
			continue
		}
		arry = append(arry, string(str))
	}
	log.Println("g.ParseConfig ok")
	DrugFile = arry
}

func DrugConfig() []string {
	configLock.RLock()
	defer configLock.RUnlock()
	return DrugFile
}
