package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestGetText(t *testing.T) {
	code, text, err := Get("http://baidu.com").Text()
	if err != nil {
		t.Errorf("api.Get failed: %d, %s", code, err.Error())
	}

	log.Printf("api.Get (%s)", text)
}

func TestGetJSON(t *testing.T) {
	agent := Get("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential")
	agent.QueryAdd("appid", "appid")
	agent.QueryAdd("secret", "secret")

	log.Printf("agent url query: (%v)", agent.QueryGet())

	type Token struct {
		Code     int64     `json:"errcode"`
		Msg      string    `json:"errmsg"`
		Secret   string    `json:"access_token"`
		ExpireIn int64     `json:"expires_in"`
		CreateAt time.Time `json:"-"`
	}

	var tk Token

	code, err := agent.JSON(&tk)
	if err != nil {
		t.Errorf("api.JSON failed: %d, %s", code, err.Error())
	}

	log.Printf("api.JSON (%v)", tk)
}

func TestProcessor(t *testing.T) {
	agent := Get("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential")
	p := func(resp *http.Response) (*http.Response, error) {
		return resp, errors.New("processor error")
	}
	agent.ResponseProcessor(p)

	rsp, err := agent.Do()
	fmt.Println("response: ", rsp)
	if err != nil {
		t.Errorf("error : %v", err)
	}
}
