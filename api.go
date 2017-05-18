package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"encoding/json"
	"encoding/xml"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

const (
	POST   = "POST"
	GET    = "GET"
	HEAD   = "HEAD"
	PUT    = "PUT"
	DELETE = "DELETE"
	PATCH  = "PATCH"
)

var types = map[string]string{
	"html":       "text/html",
	"json":       "application/json",
	"xml":        "application/xml",
	"text":       "text/plain",
	"urlencoded": "application/x-www-form-urlencoded",
	"form":       "application/x-www-form-urlencoded",
	"form-data":  "application/x-www-form-urlencoded",
}

type Agent struct {
	u       *url.URL
	t       string
	m       string
	heads   map[string]string
	query   url.Values
	cookies []*http.Cookie
	data    io.Reader
	length  int
	Error   error
	debug   bool
	conn    *http.Client
}

func URL(aurl string) *Agent {
	u, err := url.Parse(aurl)
	return &Agent{
		u:       u,
		t:       types["html"],
		m:       GET,
		heads:   make(map[string]string),
		query:   url.Values{},
		cookies: make([]*http.Cookie, 0),
		Error:   err,
		conn:    http.DefaultClient,
	}
}

func Get(aurl string) *Agent {
	return URL(aurl).Method(GET)
}

func Post(aurl string) *Agent {
	return URL(aurl).Method(POST)
}

func Patch(aurl string) *Agent {
	return URL(aurl).Method(PATCH)
}

func Put(aurl string) *Agent {
	return URL(aurl).Method(PUT)
}

func Head(aurl string) *Agent {
	return URL(aurl).Method(HEAD)
}

func HTTP(host string) *Agent {
	return URL(fmt.Sprintf("http://%s", host))
}

func HTTPs(host string) *Agent {
	return URL(fmt.Sprintf("https://%s", host))
}

func (a *Agent) Transport(tr http.RoundTripper) *Agent {
	a.conn = &http.Client{
		Transport: tr,
	}
	return a
}

func (a *Agent) Debug(flag bool) *Agent {
	a.debug = flag
	return a
}
func (a *Agent) URI(uri string) *Agent {
	a.u.Path = uri
	return a
}

func (a *Agent) QueryGet() url.Values {
	q := a.u.Query()
	for k, v := range a.query {
		for _, vv := range v {
			q.Add(k, vv)
		}
	}
	return q
}

func (a *Agent) QuerySet(key string, value string) *Agent {
	a.query.Set(key, value)
	return a
}

func (a *Agent) QueryAdd(key string, value string) *Agent {
	a.query.Add(key, value)
	return a
}

func (a *Agent) Fragment(value string) *Agent {
	a.u.Fragment = value
	return a
}

func (a *Agent) QueryDel(key string) *Agent {
	a.query.Del(key)
	return a
}

func (a *Agent) HeadSet(key string, value string) *Agent {
	a.heads[key] = value
	return a
}

func (a *Agent) HeadDel(key string) *Agent {
	delete(a.heads, key)
	return a
}

func (a *Agent) BasicAuthSet(user, password string) *Agent {
	a.u.User = url.UserPassword(user, password)
	return a
}

func (a *Agent) BasicAuthDel() *Agent {
	a.u.User = nil
	return a
}

func (a *Agent) CookiesAdd(cookies ...*http.Cookie) *Agent {
	a.cookies = append(a.cookies, cookies...)
	return a
}

func (a *Agent) Method(m string) *Agent {
	a.m = m
	return a
}

func (a *Agent) ContentType(t string) *Agent {
	if ct, ok := types[t]; ok {
		a.m = ct
	}
	return a
}

func (a *Agent) FormData(form map[string][]string) *Agent {
	data := url.Values(form).Encode()
	a.data = strings.NewReader(data)
	a.length = len(data)
	return a
}

func JSONMarshal(v interface{}, unescape bool) ([]byte, error) {
	b, err := json.Marshal(v)

	if unescape {
		b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
		b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
		b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	}
	return b, err
}

func (a *Agent) JSONData(args ...interface{}) *Agent {
	if len(args) == 1 {
		data, err := JSONMarshal(args[0], false)
		a.data = bytes.NewBuffer(data)
		a.length = len(data)
		a.Error = err
	}

	if len(args) == 2 {
		data, err := JSONMarshal(args[0], args[1].(bool))
		a.data = bytes.NewBuffer(data)
		a.length = len(data)
		a.Error = err
	}
	return a
}

func (a *Agent) XMLData(obj interface{}) *Agent {
	data, err := xml.Marshal(obj)
	a.data = bytes.NewBuffer(data)
	a.length = len(data)
	a.Error = err
	return a
}

func (a *Agent) Do() (*http.Response, error) {
	if a.Error != nil {
		return nil, a.Error
	}
	req, err := http.NewRequest(a.m, a.u.String(), a.data)
	if err != nil {
		a.Error = err
		return nil, err
	}

	//! headers
	req.Header.Set("Content-Type", types[a.t])
	for k, v := range a.heads {
		req.Header.Set(k, v)
	}

	//! query
	q := req.URL.Query()
	for k, v := range a.query {
		for _, vv := range v {
			q.Add(k, vv)
		}
	}
	req.URL.RawQuery = q.Encode()

	//! basic auth
	if a.u.User != nil {
		if password, ok := a.u.User.Password(); ok {
			req.SetBasicAuth(a.u.User.Username(), password)
		}
	}

	//! cookies
	for _, cookie := range a.cookies {
		req.AddCookie(cookie)
	}

	//! do
	if a.debug {
		dump, _ := httputil.DumpRequest(req, true)
		log.Printf("api request\n-------------------------------\n%s\n", string(dump))
	}

	return a.conn.Do(req)
}

func (a *Agent) Status() (int, string, error) {
	resp, err := a.Do()
	a.Error = err
	return resp.StatusCode, resp.Status, err
}

func (a *Agent) Bytes() (int, []byte, error) {
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return resp.StatusCode, nil, err
	}
	defer resp.Body.Close()

	if a.debug {
		dump, _ := httputil.DumpResponse(resp, true)
		log.Printf("api response\n--------------------------------\n%s\n", string(dump))
	}

	if resp.StatusCode != 200 {
		a.Error = fmt.Errorf(resp.Status)
		return resp.StatusCode, nil, a.Error
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		a.Error = err
		return resp.StatusCode, nil, err
	}

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return resp.StatusCode, body, a.Error
}

func (a *Agent) Text() (int, string, error) {
	a.t = "text"
	code, bytes, err := a.Bytes()
	return code, string(bytes), err
}

func (a *Agent) JSON(obj interface{}) (int, error) {
	a.t = "json"

	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return resp.StatusCode, err
	}
	defer resp.Body.Close()

	//! decode bytes to json
	if obj != nil {
		if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
			a.Error = err
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, a.Error
}

func (a *Agent) JSONPB(obj proto.Message) (int, error) {
	a.t = "json"
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return resp.StatusCode, err
	}
	defer resp.Body.Close()

	//! decode bytes to jsonpb
	if obj != nil {
		if err := jsonpb.Unmarshal(resp.Body, obj); err != nil {
			a.Error = err
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, a.Error
}

func (a *Agent) XML(obj interface{}) (int, error) {
	a.t = "xml"

	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return resp.StatusCode, err
	}
	defer resp.Body.Close()

	//! decode bytes to json
	if obj != nil {
		if err := xml.NewDecoder(resp.Body).Decode(&obj); err != nil {
			a.Error = err
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, a.Error
}
