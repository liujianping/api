package api

import (
	"log"
	"testing"
	"time"
)

func TestGetText(t *testing.T) {
	_, text, err := Get("http://baidu.com").Text()
	if err != nil {
		t.Errorf("api.Get failed: %s", err.Error())
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

	_, _, err := agent.JSON(&tk)
	if err != nil {
		t.Errorf("api.JSON failed: %s", err.Error())
	}

	log.Printf("api.JSON (%v)", tk)
}
