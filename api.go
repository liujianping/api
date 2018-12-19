package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
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
	"multipart":  "multipart/form-data",
}

type ResponseProcessor func(*http.Response) (*http.Response, error)

type Cipher interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type Agent struct {
	u         *url.URL
	t         string
	m         string
	prefix    string
	headerIn  http.Header
	headerOut http.Header
	query     url.Values
	cookies   []*http.Cookie
	files     []*File
	data      io.Reader
	length    int
	cipher    Cipher
	Error     error
	debug     bool
	conn      *http.Client
	processor ResponseProcessor
}

func URL(aurl string) *Agent {
	u, err := url.Parse(aurl)
	if err != nil {
		panic(err)
	}
	prefix := strings.TrimSuffix(u.Path, "/")
	return &Agent{
		u:         u,
		t:         types["html"],
		m:         GET,
		prefix:    prefix,
		headerIn:  make(map[string][]string),
		headerOut: make(map[string][]string),
		query:     url.Values{},
		cookies:   make([]*http.Cookie, 0),
		files:     make([]*File, 0),
		Error:     err,
		conn:      http.DefaultClient,
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

func (a *Agent) SetCipher(cipher Cipher) *Agent {
	a.cipher = cipher
	return a
}
func (a *Agent) ResponseProcessor(processor ResponseProcessor) *Agent {
	a.processor = processor
	return a
}

func (a *Agent) Prefix(prefix string) *Agent {
	a.prefix = strings.TrimSuffix(prefix, "/")
	return a
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
	if len(a.prefix) > 0 {
		a.u.Path = a.prefix + uri
	}
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

func (a *Agent) SetHead(hdr http.Header) *Agent {
	for k, vs := range hdr {
		for _, v := range vs {
			a.headerIn.Add(k, v)
		}
	}
	return a
}

func (a *Agent) HeadSet(key string, value string) *Agent {
	a.headerIn.Set(key, value)
	return a
}

func (a *Agent) HeadAdd(key string, value string) *Agent {
	a.headerIn.Add(key, value)
	return a
}

func (a *Agent) HeadDel(key string) *Agent {
	a.headerIn.Del(key)
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
	a.t = "form"
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
	a.t = "json"
	return a
}

func (a *Agent) PBData(obj proto.Message) *Agent {
	buf := bytes.NewBuffer([]byte{})
	marshaler := &jsonpb.Marshaler{EmitDefaults: true}
	err := marshaler.Marshal(buf, obj)
	a.data = buf
	a.Error = err
	a.length = buf.Len()
	a.t = "json"
	return a
}

func (a *Agent) XMLData(obj interface{}) *Agent {
	data, err := xml.Marshal(obj)
	a.data = bytes.NewBuffer(data)
	a.length = len(data)
	a.Error = err
	a.t = "xml"
	return a
}

type File struct {
	Filename  string
	Fieldname string
	Data      []byte
}

func NewFile(field string, filename string) (*File, error) {
	absFile, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	fn := filepath.Base(absFile)
	data, err := ioutil.ReadFile(absFile)
	if err != nil {
		return nil, err
	}
	return &File{
		Filename:  fn,
		Fieldname: field,
		Data:      data,
	}, nil
}

func NewFileByBytes(field string, filename string, data []byte) (*File, error) {
	fn := filepath.Base(filename)
	return &File{
		Filename:  fn,
		Fieldname: field,
		Data:      data,
	}, nil
}

func NewFileByReader(field string, filename string, rd io.Reader) (*File, error) {
	fn := filepath.Base(filename)
	data, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	return &File{
		Filename:  fn,
		Fieldname: field,
		Data:      data,
	}, nil
}

func (a *Agent) FileData(files ...*File) *Agent {
	a.files = append(a.files, files...)
	a.t = "multipart"
	return a
}

func (a *Agent) Do() (*http.Response, error) {
	if a.Error != nil {
		return nil, a.Error
	}

	content_type := types[a.t]
	if len(a.files) > 0 {
		buf := &bytes.Buffer{}
		mw := multipart.NewWriter(buf)

		for _, file := range a.files {
			fw, _ := mw.CreateFormFile(file.Fieldname, file.Filename)
			fw.Write(file.Data)
		}
		a.data = buf
		content_type = mw.FormDataContentType()
		mw.Close()
	}

	req, err := http.NewRequest(a.m, a.u.String(), a.data)
	if err != nil {
		a.Error = err
		return nil, err
	}

	//! headers
	req.Header = a.headerIn
	req.Header.Set("Content-Type", content_type)

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

	//! cipher
	if a.cipher != nil {
		byts, err := ioutil.ReadAll(a.data)
		if err != nil {
			return nil, err
		}
		enbyts, err := a.cipher.Encrypt(byts)
		if err != nil {
			return nil, err
		}
		a.data = bytes.NewBuffer(enbyts)
		a.length = len(enbyts)
	}

	resp, err := a.conn.Do(req)
	if resp != nil {
		a.headerOut = resp.Header
	}

	//! cipher
	if a.cipher != nil {
		if strings.ToLower(resp.Header.Get("X-CIPHER-ENCODED")) == "true" {
			enbyts, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			debyts, err := a.cipher.Decrypt(enbyts)
			if err != nil {
				return nil, err
			}
			resp.Header.Del("X-CIPHER-ENCODED")
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(debyts))
			resp.ContentLength = int64(len(debyts))
		}
	}

	if a.debug {
		dump, _ := httputil.DumpResponse(resp, true)
		log.Printf("api response\n-------------------------------\n%s\n", string(dump))
	}

	//response processor
	if a.processor != nil && err == nil {
		return a.processor(resp)
	}
	return resp, err
}

func (a *Agent) Status() (int, string, error) {
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err
	}
	return resp.StatusCode, resp.Status, nil
}

func (a *Agent) Bytes() (int, []byte, error) {
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return http.StatusInternalServerError, []byte{}, err
	}
	defer resp.Body.Close()

	if a.debug {
		dump, _ := httputil.DumpResponse(resp, true)
		log.Printf("api response\n--------------------------------\n%s\n", string(dump))
	}

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			a.Error = err
			return resp.StatusCode, nil, fmt.Errorf(resp.Status)
		}
		a.Error = fmt.Errorf(string(body))
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
	code, bytes, err := a.Bytes()
	return code, string(bytes), err
}

func (a *Agent) JSON(obj interface{}) (int, error) {
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			a.Error = err
			return resp.StatusCode, fmt.Errorf(resp.Status)
		}
		a.Error = fmt.Errorf(resp.Status)
		return resp.StatusCode, fmt.Errorf(string(body))
	}

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
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			a.Error = err
			return resp.StatusCode, fmt.Errorf(resp.Status)
		}
		a.Error = fmt.Errorf(resp.Status)
		return resp.StatusCode, fmt.Errorf(string(body))
	}

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
	resp, err := a.Do()
	if err != nil {
		a.Error = err
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			a.Error = err
			return resp.StatusCode, fmt.Errorf(resp.Status)
		}
		a.Error = fmt.Errorf(resp.Status)
		return resp.StatusCode, fmt.Errorf(string(body))
	}

	//! decode bytes to json
	if obj != nil {
		if err := xml.NewDecoder(resp.Body).Decode(&obj); err != nil {
			a.Error = err
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, a.Error
}

func (a *Agent) GetHeadIn() http.Header {
	return a.headerIn
}

func (a *Agent) GetHeadOut() http.Header {
	return a.headerOut
}
