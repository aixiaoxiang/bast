// Copyright 2018 The axx Authors. All rights reserved.

package bast

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/aixiaoxiang/bast/guid"
	"github.com/aixiaoxiang/bast/ids"
	"github.com/aixiaoxiang/bast/logs"
	"github.com/aixiaoxiang/bast/session"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

//const code
const (
	SerError                = 0       // error code
	SerOK                   = 1       // ok code
	SerDBError              = -10000  // db error code
	SerNoDataError          = -20000  // no data error code
	SerSignOutError         = -30000  // user sign out error code
	SerUserNotExistError    = -40000  // user not exist code
	SerInvalidParamError    = -50000  // invalid param  code
	SerInvalidUserAuthorize = -60000  // invalid user authorize  code
	SerExist                = -70000  // exist code
	SerNotExist             = -80000  // not exist code
	SerTry                  = -99999  // please try code
	SerMustFailed           = -111111 // must failed code
	SerFailed               = -222222 // failed code
	SerAuthorizationFailed  = -888888 // authorization failed code
)

//Context is app Context
type Context struct {
	//In A Request represents an HTTP request received by a server
	// or to be sent by a client.
	In *http.Request
	//Out A ResponseWriter interface is used by an HTTP handler to
	// construct an HTTP response.
	Out http.ResponseWriter
	//Params httprouter Params,/:name/:age
	Params httprouter.Params
	//isParseForm Parse tag
	isParseForm bool
	//Authorization is need authorization
	Authorization bool
	//IsAuthorization is authorization finish?
	IsAuthorization bool
	//Session is session
	Session session.Store
}

//Msgs is response message
type Msgs struct {
	Code int    `gorm:"-" json:"code"`
	Msg  string `gorm:"-" json:"msg"`
}

//MsgDetail is response detail message
type MsgDetail struct {
	Code   int    `gorm:"-" json:"code"`
	Msg    string `gorm:"-" json:"msg"`
	Detail string `gorm:"-" json:"detail"`
}

//Data is response data
type Data struct {
	Msgs `gorm:"-"`
	Data interface{} `gorm:"-"  json:"data"`
}

//DataPage is Pagination data
type DataPage struct {
	Msgs
	Data  interface{} `gorm:"-"  json:"data"`
	Page  int         `gorm:"-"  json:"page"`
	Total int         `gorm:"-"  json:"total"`
}

//InfoPage is invalid Pagination data
type InfoPage struct {
	DataPage
	Invalid bool `gorm:"-"  json:"invalid"`
	Fix     bool `gorm:"-"  json:"fix"`
}

//JSON  output JSON Data to client
//v data
func (c *Context) JSON(v interface{}) {
	c.JSONWithCodeMsg(v, SerOK, "")
}

//JSONWithCode output JSON Data to client
//v data
//code is message code
func (c *Context) JSONWithCode(v interface{}, code int) {
	c.JSONWithCodeMsg(v, code, "")
}

//JSONWithMsg output JSON Data to client
//v data
//msg is string message
func (c *Context) JSONWithMsg(v interface{}, msg string) {
	c.JSONWithCodeMsg(v, SerOK, msg)
}

//JSONWithCodeMsg output JSON Data to client
//v data
//code is message code
//msg is string message
func (c *Context) JSONWithCodeMsg(v interface{}, code int, msg string) {
	_, isData := v.(*Data)
	if !isData {
		_, isData = v.(*Msgs)
	}
	if !isData {
		_, isData = v.(*DataPage)
	}
	if !isData {
		d := &Data{}
		d.Code = code
		d.Msg = msg
		d.Data = v
		c.JSONResult(d)
		d.Data = nil
		d = nil
	} else {
		c.JSONResult(v)
	}
}

//JSONWithPage output Pagination JSON Data to client
func (c *Context) JSONWithPage(v interface{}, page, total int) {
	c.JSONWithPageAndCodeMsg(v, page, total, SerOK, "")
}

//JSONWithPageAndCode output Pagination JSON Data to client
func (c *Context) JSONWithPageAndCode(v interface{}, page, total, code int, msg string) {
	c.JSONWithPageAndCodeMsg(v, page, total, code, msg)
}

//JSONWithPageAndCodeMsg output Pagination JSON Data to client
func (c *Context) JSONWithPageAndCodeMsg(v interface{}, page, total, code int, msg string) {
	d := &InfoPage{}
	_, _total, pageRow := c.Page()
	if _total == 0 {
		last := int(math.Ceil(float64(total) / float64(pageRow)))
		if page >= last {
			page = last - 1
			d.Fix = true
		}
	}
	page++
	if v != nil {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Array:
		case reflect.Map:
		case reflect.Slice:
			s := reflect.ValueOf(v)
			if s.Len() == 0 {
				d.Invalid = true
			}
			break
		}
	} else {
		d.Invalid = true
	}

	d.Data = v
	d.Page = page
	d.Total = total
	d.Code = code
	d.Msg = msg
	if d.Invalid || d.Fix {
		c.JSONResult(d)
	} else {
		c.JSONResult(d.DataPage)
	}
	d.Data = nil
	d = nil
}

//JSONResult output Data to client
func (c *Context) JSONResult(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		logs.Info("JSONResult-Error=" + err.Error())
		c.StatusCode(http.StatusInternalServerError)
		return
	}
	c.Out.Header().Set("Content-Type", "application/json")
	c.Out.Write(data)
	data = nil
}

//Success output success result to client
//	msg is success message
func (c *Context) Success(msg string) {
	d := &Msgs{}
	d.Code = SerOK
	d.Msg = msg
	c.JSON(d)
	d = nil
}

//Successf output success result and format to client
func (c *Context) Successf(format string, a ...interface{}) {
	if a != nil && len(a) > 0 {
		format = fmt.Sprintf(format, a...)
	}
	d := &Msgs{}
	d.Code = SerOK
	d.Msg = format
	c.JSON(d)
	d = nil
}

//Failed  output failed result to client
//param:
//	msg is fail message
//	err error
func (c *Context) Failed(msg string, err ...error) {
	c.FailResult(msg, SerError, err...)
}

//Faileds  output failed detail result to client
//param:
//	msg is fail message
//	detail is detail message
func (c *Context) Faileds(msg string, detail string) {
	d := &MsgDetail{}
	d.Code = SerError
	d.Msg = msg
	d.Detail = detail
	c.JSONWithCode(d, SerError)
	d = nil
}

//Failedf output failed result and format to client
func (c *Context) Failedf(format string, a ...interface{}) {
	var err error
	if a != nil {
		lg := len(a)
		if lg > 0 {
			if a[lg-1] != nil {
				err, _ = a[lg-1].(error)
			}
			if err != nil {
				a = a[0 : lg-1]
			}
			if len(a) > 0 {
				format = fmt.Sprintf(format, a...)
			}
		}
	}
	c.FailResult(format, SerError, err)
}

//Result  output result to client
//param:
//	msg is fail message
//	detail is detail message
func (c *Context) Result(msg string, detail ...string) {
	d := &MsgDetail{}
	d.Code = SerOK
	d.Msg = msg
	if detail != nil {
		for _, s := range detail {
			if d.Detail != "" {
				d.Detail += ","
			}
			d.Detail += s
		}
	}
	c.JSONWithCode(d, SerError)
	d = nil
}

//SignOutError output user signout to client
//param:
//	msg message
func (c *Context) SignOutError(msg string) {
	c.FailResult(msg, SerSignOutError)
}

//DBError  output dataBase error to client
//param:
//	err db.error
func (c *Context) DBError(err error) {
	msg := "DataBase error"
	if err != nil {
		msg = "DataBase error,detail：" + err.Error()
	} else {
		msg = "DataBase error"
	}
	c.FailResult(msg, SerDBError)
}

//FailResult output fail result to client
//param:
//	msg failed message
//	errCode ailed message code
//  err  error
func (c *Context) FailResult(msg string, errCode int, err ...error) {
	d := &Msgs{}
	if errCode == 0 {
		errCode = SerError
	}
	d.Code = errCode
	d.Msg = msg
	if err != nil && err[0] != nil {
		d.Msg += ", [" + err[0].Error() + "]"
	}
	c.JSONWithCode(d, errCode)
	d = nil
}

//NoData output no data result to client
//param:
//	err message
func (c *Context) NoData(msg ...string) {
	msgs := "Sorry！No Data"
	if msg == nil {
		msgs = "Sorry！No Data"
	} else {
		msgs = msg[0]
	}
	c.FailResult(msgs, SerNoDataError)
}

//Say output raw bytes to client
//param:
//	data raw bytes
func (c *Context) Say(data []byte) {
	c.Out.Write(data)
}

//SayStr output string to client
//param:
//	str string
func (c *Context) SayStr(str string) {
	c.Out.Write([]byte(str))
}

//SendFile send file to client
//param:
//	fileName is file name
//  rawFileName is raw file name
func (c *Context) SendFile(fileName string, rawFileName ...string) {
	dir := filepath.Dir(fileName)
	fileName = filepath.Base(fileName)
	url := c.BaseURL("f/" + fileName)
	fileName = "/f/" + fileName
	fs := http.StripPrefix("/f/", http.FileServer(http.Dir(dir)))
	r, _ := http.NewRequest("GET", url, nil)
	raw := fileName
	if rawFileName != nil {
		raw = rawFileName[0]
		c.Out.Header().Set("Content-Disposition", "attachment; filename="+raw)
	}
	fs.ServeHTTP(c.Out, r)
	r = nil
	fs = nil
}

//JSONToStr JSON to string
//param:
//	obj is object
func (c *Context) JSONToStr(obj interface{}) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", nil
	}
	return string(data), err
}

//StrToJSON string to JSON
//param:
//	str json string
//  obj is object
func (c *Context) StrToJSON(str string, obj interface{}) error {
	return c.JSONDecode(strings.NewReader(str), obj)
}

//StatusCode set current request statusCode
//param:
//	statusCode HTTP status code. such as: 200x,300x and so on
func (c *Context) StatusCode(statusCode int) {
	c.Out.WriteHeader(statusCode)
	c.Out.Write([]byte(http.StatusText(statusCode)))
}

//GetRawStr getter raw string value from current request(request body)
func (c *Context) GetRawStr() string {
	body, err := ioutil.ReadAll(c.In.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

//GetString  gets a string value from  the current request  based on the key
//param:
//	key is key name
func (c *Context) GetString(key string) string {
	d := c.GetStrings(key)
	if len(d) > 0 {
		return d[0]
	}
	return ""
}

//GetTrimString  Use the key to get a non-space string value from the current request
//param:
//	key is key name
func (c *Context) GetTrimString(key string) string {
	return strings.TrimSpace(c.GetString(key))
}

//GetStringSlice Use the key to get all string value from the current request
//param:
//	key is key name
//  sep spilt char
func (c *Context) GetStringSlice(key, sep string) *[]string {
	d := c.GetStrings(key)
	if len(d) > 0 {
		s := strings.Split(d[0], sep)
		return &s
	}
	return nil
}

//GetIntSlice Use the key to get all int value from the current request
//param:
//	key is key name
//  sep spilt char
func (c *Context) GetIntSlice(key, sep string) *[]int64 {
	d := c.GetStrings(key)
	if len(d) > 0 {
		s := strings.Split(d[0], sep)
		lg := len(s)
		si := make([]int64, lg, lg)
		for i := 0; i < lg; i++ {
			si[i], _ = strconv.ParseInt(s[i], 10, 64)
		}
		return &si
	}
	return nil
}

//GetIntSliceAndRemovePrefix Use the key to get all int value from the current request and remove prefix of each
//param:
//	key is key name
//  sep spilt char
//  prefix	remove prefix string
func (c *Context) GetIntSliceAndRemovePrefix(key, sep, prefix string) (*[]int64, bool) {
	d := c.GetStrings(key)
	has := false
	if len(d) > 0 {
		ss := d[0]
		if prefix != "" {
			has = strings.HasPrefix(ss, prefix)
			ss = strings.TrimPrefix(d[0], prefix)
		}
		s := strings.Split(ss, sep)
		lg := len(s)
		si := make([]int64, 0, lg)
		for i := 0; i < lg; i++ {
			n, err := strconv.ParseInt(s[i], 10, 64)
			if err == nil {
				si = append(si, n)
			}
		}
		if len(si) > 0 {
			return &si, has
		}
	}
	return nil, false
}

//GetParam  Use the key to get all int value from the current request url
//note：xx/:name/:name2
//param:
//	key key name
func (c *Context) GetParam(key string) string {
	return c.Params.ByName(key)
}

//GetLeftLikeString get a sql(left like 'xx%') string value from the current request  based on the key
//param:
//	key is key name
func (c *Context) GetLeftLikeString(key string) string {
	d := c.GetStrings(key)
	if len(d) > 0 {
		r := d[0]
		if r != "" {
			return r + "%"
		}
	}
	return ""
}

//GetRightLikeString get a sql(right like '%xx') string value from the current request  based on the key
//param:
//	key is key name
func (c *Context) GetRightLikeString(key string) string {
	d := c.GetStrings(key)
	if len(d) > 0 {
		r := d[0]
		if r != "" {
			return "%" + r
		}
	}
	return ""
}

//GetLikeString  get a sql(like '%xx%') string value from the current request  based on the key
//param:
//	key is key name
func (c *Context) GetLikeString(key string) string {
	d := c.GetStrings(key)
	if len(d) > 0 {
		r := d[0]
		if r != "" {
			return "%" + r + "%"
		}
	}
	return ""
}

//GetBool get a bool value  from the current request  based on the key
//param:
//	key is key name
func (c *Context) GetBool(key string) bool {
	d := c.GetStrings(key)
	if len(d) > 0 {
		ok, err := strconv.ParseBool(d[0])
		if err == nil {
			return ok
		}
	}
	return false
}

//GetBoolWithDefault get a bool value from the current request  based on the key
//param:
//	key is key name
//  def is default value
func (c *Context) GetBoolWithDefault(key string, def bool) bool {
	d := c.GetStrings(key)
	if len(d) > 0 {
		ok, err := strconv.ParseBool(d[0])
		if err == nil {
			return ok
		}
	}
	return def
}

//GetStrings gets strings from the current request based on the key
//param:
//	key is key name
func (c *Context) GetStrings(key string) []string {
	c.ParseForm()
	return c.In.Form[key]
}

//HasParam has a param from the current request based on the key(May not have a value)
//param:
//	key is key name
func (c *Context) HasParam(key string) bool {
	c.ParseForm()
	_, ok := c.In.Form[key]
	return ok
}

//Form gets all form params from the current(uri not included)
func (c *Context) Form() url.Values {
	c.ParseForm()
	return c.In.Form
}

//PostForm gets all form params from the current(uri and form)
func (c *Context) PostForm() url.Values {
	c.ParseForm()
	return c.In.PostForm
}

//Query gets all query params from the current request url
func (c *Context) Query() url.Values {
	c.ParseForm()
	return c.In.URL.Query()
}

//GetInt gets a int value from the current request  based on the key
//param:
//	key is key name
//	def default value
func (c *Context) GetInt(key string, def ...int) (int, error) {
	d := c.GetString(key)
	v, err := strconv.Atoi(d)
	if err != nil {
		if def != nil && len(def) > 0 {
			v = def[0]
		} else {
			v = 0
		}
	}
	return v, err
}

//GetIntVal gets a int value  from the current request  based on the key（errors not included）
//param:
//	key is key name
//	def default value
func (c *Context) GetIntVal(key string, def ...int) int {
	d := c.GetString(key)
	v, err := strconv.Atoi(d)
	if err != nil {
		if def != nil && len(def) > 0 {
			v = def[0]
		} else {
			v = 0
		}
	}
	return v
}

//GetInt64 gets a int64 value  from the current request url  based on the key
//param:
//	key is key name
//	def default value
func (c *Context) GetInt64(key string, def ...int64) (int64, error) {
	d := c.GetString(key)
	v, err := strconv.ParseInt(d, 10, 64)
	if err != nil {
		if def != nil && len(def) > 0 {
			v = def[0]
		} else {
			v = 0
		}
	}
	return v, err
}

//GetInt64Val gets a int64 value  from the current request  based on the key（errors not included）
//param:
//	key is key name
//	def default value
func (c *Context) GetInt64Val(key string, def ...int64) int64 {
	d := c.GetString(key)
	v, err := strconv.ParseInt(d, 10, 64)
	if err != nil {
		if def != nil && len(def) > 0 {
			v = def[0]
		} else {
			v = 0
		}
	}
	return v
}

//GetFloat gets a float value  from the current request uri  based on the key
//param:
//	key is key name
//	def default value
func (c *Context) GetFloat(key string, def ...float64) (float64, error) {
	d := c.GetString(key)
	v, err := strconv.ParseFloat(d, 64)
	if err != nil {
		if def != nil && len(def) > 0 {
			v = def[0]
		} else {
			v = 0
		}
	}
	return v, err
}

//Page get pages param from the current request
//param:
//	page 	current page index(start 1)
//	total 	all data total count(cache total count for first service return)
//  pageRow page maximum size(default is 100 row)
func (c *Context) Page() (page int, total int, pageRow int) {
	page, _ = c.GetInt("page")
	total, _ = c.GetInt("total")
	pageRow, _ = c.GetInt("pageRow", 100)
	if page > 0 {
		page--
	}
	if pageRow > 100 {
		pageRow = 100
	} else if pageRow <= 0 {
		pageRow = 100
	}
	return page, total, pageRow
}

//Pages get pages param from the current request and check last page
//param:
//	page 	 current page index(start 1)
//	total 	 all data total count(cache total count for first service return)
//  pageRow  page maximum size(default is 100 row)
func (c *Context) Pages() (page, total, pageRow int) {
	page, total, pageRow = c.Page()
	if total > 0 {
		last := int(math.Ceil(float64(total) / float64(pageRow)))
		if page >= last {
			total = 0
		}
	}
	return page, total, pageRow
}

//Offset return page offset
//param:
//	total 	all data total count(cache total count for first service return)
func (c *Context) Offset(total int) int {
	page, _total, pageRow := c.Page()
	if _total == 0 {
		last := int(math.Ceil(float64(total) / float64(pageRow)))
		if page >= last {
			page = last - 1
		}
	}
	// fixPage := c.HasParam("fixPage")
	// if fixPage {
	// 	last := int(math.Ceil(float64(total) / float64(pageRow)))
	// 	if page >= last {
	// 		page = last - 1
	// 	}
	// }
	offset := page * pageRow
	return offset
}

//JSONObj gets data from the current request body(JSON fromat) and convert it to a objecet
//param:
//	obj target object
func (c *Context) JSONObj(obj interface{}) error {
	return c.JSONDecode(c.In.Body, obj)
}

//GetJSON gets data from the current request body(JSON fromat) and convert it to a objecet
//param:
//	obj target object
func (c *Context) GetJSON(obj interface{}) error {
	return c.JSONObj(obj)
}

//JSONDecode gets data from the r reader(JSON fromat) and convert it to a objecet
//param:
//	r is a reader
//	obj target object
func (c *Context) JSONDecode(r io.Reader, obj interface{}) error {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	// logs.Debug("JSONDecode=" + string(body))
	err = json.Unmarshal(body, obj)
	if err != nil {
		if app.Debug {
			logs.Debug("JSONDecode-Error=" + err.Error() + ",detail=" + string(body))
		} else {
			logs.Debug("JSONDecode-Error=" + err.Error())
		}
		body = nil
		return err
	}
	return err
}

//XMLObj gets data from the current request(xml format) and convert it to a object
//param:
//	obj target object
func (c *Context) XMLObj(obj interface{}) error {
	return c.XMLDecode(c.In.Body, obj)
}

//GetXML gets data from the current request body(xml format)  and  convert it to a object
//param:
//	obj target object
func (c *Context) GetXML(obj interface{}) error {
	return c.XMLObj(obj)
}

//XMLDecode  gets data from the r reader(xml format) and convert it to a object
//param:
//	r is a reader
//	obj target object
func (c *Context) XMLDecode(r io.Reader, obj interface{}) error {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = xml.Unmarshal(body, obj)
	if err != nil {
		if app.Debug {
			logs.Debug("XMLDecode-Err=" + err.Error() + ",detail=" + string(body))
		} else {
			logs.Debug("XMLDecode-Err=" + err.Error())
		}
		body = nil
		return err
	}
	return err
}

//MapObj gets current request body and convert it to a map
func (c *Context) MapObj() map[string]interface{} {
	body, _ := ioutil.ReadAll(c.In.Body)
	result := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &result)
	if err == nil {
		return result
	}
	return nil
}

// ParseForm populates r.Form and r.PostForm.
//
// For all requests, ParseForm parses the raw query from the URL and updates
// r.Form.
//
// For POST, PUT, and PATCH requests, it also parses the request body as a form
// and puts the results into both r.PostForm and r.Form. Request body parameters
// take precedence over URL query string values in r.Form.
//
// For other HTTP methods, or when the Content-Type is not
// application/x-www-form-urlencoded, the request Body is not read, and
// r.PostForm is initialized to a non-nil, empty value.
//
// If the request Body's size has not already been limited by MaxBytesReader,
// the size is capped at 10MB.
//
// ParseMultipartForm calls ParseForm automatically.
// ParseForm is idempotent.
func (c *Context) ParseForm() {
	if !c.isParseForm {
		c.In.ParseForm()
		c.isParseForm = true
	}
}

// ParseMultipartForm parses a request body as multipart/form-data.
// The whole request body is parsed and up to a total of maxMemory bytes of
// its file parts are stored in memory, with the remainder stored on
// disk in temporary files.
// ParseMultipartForm calls ParseForm if necessary.
// After one call to ParseMultipartForm, subsequent calls have no effect.
func (c *Context) ParseMultipartForm(maxMemory int64) error {
	return c.In.ParseMultipartForm(maxMemory)
}

//SessionRead get session value by key
func (c *Context) SessionRead(key string) interface{} {
	if c.Session != nil {
		return c.Session.Get(key)
	}
	return nil
}

//SessionWrite set session value by key
func (c *Context) SessionWrite(key string, value interface{}) error {
	if c.Session != nil {
		return c.Session.Set(key, value)
	}
	return nil
}

//SessionDelete delete session value by key
func (c *Context) SessionDelete(key string) error {
	if c.Session != nil {
		return c.Session.Delete(key)
	}
	return nil
}

//SessionClear delete all session
func (c *Context) SessionClear() error {
	if c.Session != nil {
		return c.Session.Clear()
	}
	return nil
}

//SessionID get sessionID
func (c *Context) SessionID() string {
	if c.Session != nil {
		return c.Session.ID()
	}
	return ""
}

//URL get eequest url
func (c *Context) URL() string {
	return strings.Join([]string{c.BaseURL(), c.In.RequestURI}, "")
}

//DefaultFileURL returns full file url
//param:
//	url is relative path
func (c *Context) DefaultFileURL(url string) string {
	if url != "" {
		if url[0] != 'f' {
			url = "f/" + url
		}
		baseURL := c.In.Header.Get("BaseUrl")
		if baseURL != "" {
			return baseURL + url
		}
		return c.baseURL() + "/" + url
	}
	return ""
}

//BaseURL gets root url(scheme+host) from current request
//param:
//	url relative path
func (c *Context) BaseURL(url ...string) string {
	baseURL := c.In.Header.Get("BaseUrl")
	if baseURL != "" {
		return baseURL + strings.Join(url, "")
	}
	return c.baseURL() + "/" + strings.Join(url, "")
}

//baseURL gets root url(scheme+host) from current request
func (c *Context) baseURL() string {
	scheme := "http://"
	if c.In.TLS != nil {
		scheme = "https://"
	}
	return strings.Join([]string{scheme, c.In.Host}, "")
}

//ClientIP return request client ip
func (c *Context) ClientIP() string {
	ps := c.Proxys()
	if len(ps) > 0 && ps[0] != "" {
		realIP, _, err := net.SplitHostPort(ps[0])
		if err != nil {
			realIP = ps[0]
		}
		return realIP
	}
	if ip, _, err := net.SplitHostPort(c.In.RemoteAddr); err == nil {
		return ip
	}
	return c.In.RemoteAddr
}

// Proxys return request proxys
// if request header has X-Real-IP, return it
// if request header has X-Forwarded-For, return it
func (c *Context) Proxys() []string {
	if v := c.In.Header.Get("X-Real-IP"); v != "" {
		return strings.Split(v, ",")
	}
	if v := c.In.Header.Get("X-Forwarded-For"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{}
}

//Redirect redirect
func (c *Context) Redirect(url string) {
	http.Redirect(c.Out, c.In, url, http.StatusFound)
}

//TemporaryRedirect redirect(note: 307 redirect，Can avoid data loss after POST redirection)
func (c *Context) TemporaryRedirect(url string) {
	http.Redirect(c.Out, c.In, url, http.StatusTemporaryRedirect)
}

//Reset current context to pool
func (c *Context) Reset() {
	c.isParseForm = false
	c.In = nil
	c.Out = nil
	c.Params = nil
	c.isParseForm = false
	c.Authorization = false
	c.IsAuthorization = false
	c.Session = nil
}

//I  log info
func (c *Context) I(msg string, fields ...zap.Field) {
	logs.I(msg, fields...)
}

//D log debug
func (c *Context) D(msg string, fields ...zap.Field) {
	logs.D(msg, fields...)
}

//E log error
func (c *Context) E(msg string, fields ...zap.Field) {
	logs.E(msg, fields...)
}

//Err log error
func (c *Context) Err(msg string, err error) {
	if msg == "" {
		msg = "error"
	}
	if err != nil {
		msg += "," + err.Error()
	}
	logs.E(msg)
}

//ID return a ID
func (c *Context) ID() int64 {
	return ids.ID()
}

//GUID return a GUID
func (c *Context) GUID() string {
	return guid.GUID()
}

// Exist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (c *Context) Exist(path string) bool {
	return PathExist(path)
}
