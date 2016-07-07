package api

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"encoding/json"
	"encoding/xml"
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
	return &Agent{
		u: &url.URL{
			Scheme: "http",
			Host:   host,
		},
		t:       types["html"],
		m:       GET,
		heads:   make(map[string]string),
		query:   url.Values{},
		cookies: make([]*http.Cookie, 0),
	}
}

func HTTPs(host string) *Agent {
	return &Agent{
		u: &url.URL{
			Scheme: "https",
			Host:   host,
		},
		t:       types["html"],
		m:       GET,
		heads:   make(map[string]string),
		query:   url.Values{},
		cookies: make([]*http.Cookie, 0),
	}
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
	a.m = types[t]
	return a
}

func (a *Agent) FormData(form map[string][]string) *Agent {
	data := url.Values(form).Encode()
	a.data = strings.NewReader(data)
	a.length = len(data)
	return a
}

func (a *Agent) JSONData(obj interface{}) *Agent {
	data, err := json.Marshal(obj)
	a.data = bytes.NewBuffer(data)
	a.length = len(data)
	a.Error = err
	return a
}

func (a *Agent) XMLData(obj interface{}) *Agent {
	data, err := xml.Marshal(obj)
	a.data = bytes.NewBuffer(data)
	a.length = len(data)
	a.Error = err
	return a
}

func (a *Agent) Bytes() (*http.Response, []byte, error) {
	req, err := http.NewRequest(a.m, a.u.String(), a.data)
	if err != nil {
		a.Error = err
		return nil, nil, err
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

	resp, err := a.conn.Do(req)
	if err != nil {
		a.Error = err
		return nil, nil, err
	}
	defer resp.Body.Close()

	if a.debug {
		dump, _ := httputil.DumpResponse(resp, true)
		log.Printf("api response\n--------------------------------\n%s\n", string(dump))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		a.Error = err
		return nil, nil, err
	}

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return resp, body, a.Error
}

func (a *Agent) Text() (*http.Response, string, error) {
	a.t = "text"

	resp, bytes, err := a.Bytes()
	return resp, string(bytes), err
}

func (a *Agent) JSON(obj interface{}) (*http.Response, []byte, error) {
	a.t = "json"

	resp, bytes, err := a.Bytes()
	if err != nil {
		return resp, bytes, err
	}

	//! decode bytes to json
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		a.Error = err
		return resp, bytes, err
	}

	return resp, bytes, a.Error
}

func (a *Agent) XML(obj interface{}) (*http.Response, []byte, error) {
	a.t = "xml"

	resp, bytes, err := a.Bytes()
	if err != nil {
		return resp, bytes, err
	}

	//! decode bytes to json
	if err := xml.NewDecoder(resp.Body).Decode(&obj); err != nil {
		a.Error = err
		return resp, bytes, err
	}

	return resp, bytes, a.Error
}
