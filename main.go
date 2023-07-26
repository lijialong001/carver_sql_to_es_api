package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/cch123/elasticsql"
	"time"
	"strconv"
	"net"
)

var (
	timeTemplate1 = "2006-01-02 15:04:05" //常规类型
	timeTemplate2 = "2006/01/02 15:04:05" //其他类型
	timeTemplate3 = "2006-01-02"          //其他类型
	timeTemplate4 = "15:04:05"            //其他类型
	timeTemplate5 = "2006-01-02 15:04"    //其他类型
)

//客户端请求信息
const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

//格式化返回信息中的data数据
type returnEsData struct {
	Table  string      `json:"table,omitempty"`
	EsData interface{} `json:"es_data,omitempty"`
}

//格式化返回信息
type returnJson struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

//日志写入函数
type Logger interface {
	Log(message string, r *http.Request)
}

type ConsoleLogger struct{}

//格式化客户端具体信息格式
type Result struct {
	Country  string  `json:"country"`
	Province string  `json:"province"`
	City     string  `json:"city"`
	District string  `json:"district"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
}

/**
 *@desc 请求入口
 *@author Carver
 */
func main() {
	http.HandleFunc("/sql_to_es", handleIndex)
	fmt.Println("Running at port 456 ...")
	err := http.ListenAndServe(":456", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

/**
 *@desc 请求路由入口
 *@params w (http.ResponseWriter)
 *@params r (*http.Request)
 *@author Carver
 */
func handleIndex(w http.ResponseWriter, r *http.Request) {
	//设置跨域
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var logger Logger
	logger = &ConsoleLogger{}
	var result returnJson

	//获取所有请求参数
	query := r.URL.Query()
	//验证参数
	sqls, ok := query["sqls"]
	if !ok || len(sqls[0]) < 1 {
		result.Code = 4002
		result.Msg = "请求参数sqls不能为空!"
	} else {
		result.Code = 200
		result.Msg = "ok"
		//处理参数
		check_sql := query["sqls"][0]
		//esType 表名 esSql es查询语句
		esSql, esType, _ := elasticsql.Convert(check_sql)
		return_all_info := returnEsData{
			Table:  esType,
			EsData: esSql,
		}
		result.Data = return_all_info
	}

	req, _ := json.Marshal(result)
	logger.Log(string(req), r)
	w.Write(req)
}

/**
 *@desc 写入日志
 *@params message (string)
 *@params r (*http.Request)
 *@return dataTime (string) 转换后的时间字符串
 *@author Carver
 */
func (cl *ConsoleLogger) Log(message string, r *http.Request) {
	ip := RemoteIp(r)
	res, _ := ip2Geo(string(ip))
	user_country := res.Country
	user_province := res.Province
	user_city := res.City
	user_lon := res.Lon //经度
	user_lat := res.Lat //纬度

	now := time.Now()
	api_time := UnixToTime(strconv.FormatInt(now.Unix(), 10))
	fmt.Println("【日志信息💻】 --------------------------------------ᕙ(▰ ‿▰ )ᕗ【start】------------------------------------------️\n")
	fmt.Println("【日志信息💻】", "[用户信息]", "⭐️用户IP:", ip, "⭐️国家:", user_country, "⭐️省份:", user_province, "⭐️城市:", user_city, "⭐️经度:", user_lon, "⭐️纬度:", user_lat, "\n")
	fmt.Println("【日志信息💻】", "[请求时间]⏰", api_time, "\n")
	fmt.Println("【日志信息💻】", "[请求结果]📁️", message, "\n")
	fmt.Println("【日志信息💻】 ️--------------------------------------ᕙ(◕ ‿ ◕)ᕗ【end】--------------------------------------------️\n")
}

/**
 *@desc 格式化时间戳
 *@params time_str 需要转化的时间戳字符串
 *@return dataTime (string) 转换后的时间字符串
 *@author Carver
 */
func UnixToTime(time_stamp_str string) (dataTime string) {
	t, _ := strconv.ParseInt(time_stamp_str, 10, 64) //外部传入的时间戳（秒为单位），必须为int64类型
	dataTime = time.Unix(t, 0).Format(timeTemplate1)
	return
}

/**
 *@desc 返回远程客户端的IP
 @return  string ip地址
 *@author Carver
 */
func RemoteIp(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}

/**
 *@desc 根据客户端的IP返回具体地区信息
 *@params ip 客户IP
 *@return res (*Result) 
 *@return err (error) 
 *@author Carver
 */
func ip2Geo(ip string) (res *Result, err error) {
	db, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ipParse := net.ParseIP(ip)
	record, err := db.City(ipParse)
	if err != nil {
		return nil, err
	}

	if record.City.Names["zh-CN"] != "" {
		res = &Result{
			Country:  record.Country.Names["zh-CN"],
			Province: record.Subdivisions[0].Names["zh-CN"],
			City:     record.City.Names["zh-CN"],
			District: "",
			Lat:      record.Location.Latitude,
			Lon:      record.Location.Longitude,
		}
	}
	return res, nil
}
