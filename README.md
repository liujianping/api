# api
===

Simplified HTTP client library :) :D

add this line for test..

##  Quick Start

````go

	import "github.com/liujianping/api"

	//! create api request agent

	agent := api.URL("http://a.domain.com/")
	agent := api.HTTP("host:port")
	agent := api.HTTPs("host")
	
	agent := api.Get("http://a.domain.com/")
	// agent := api.URL("http://a.domain.com/").Method(api.GET)

	agent := api.Post("http://a.domain.com/")
	agent := api.Patch("http://a.domain.com/")
	agent := api.Put("http://a.domain.com/")
	agent := api.Head("http://a.domain.com/")

	//! set api request method & URI & headers & parameters & form-data

	agent.Method(api.POST)

	agent.URI("/cgi/token")

	agent.HeadSet("key", "value")
	agent.HeadDel("key", "value")

	agent.QuerySet("key", "value")
	agent.QueryAdd("key", "value")
	agent.QueryDel("key", "value")

	agent.FormData(form)
	agent.JSONData(obj)
	agent.XMLData(obj)

	//! chain invoke
	var result Result{}

	if err := api.HTTP("api.demo.com:8080").URI("/a/b/c").QuerySet("x", "y").JSONData(map[string]interface{}{
		"aaa": "xxx",
		"bbb": 100,
		}).Method(api.POST).JSON(&result).Error; err != nil {
			//! do something
		}


	//! do api request

	resp, []byte, err := agent.Bytes()

	resp, string, err := agent.Text()

	resp, []byte, err := agent.JSON(&json)

	resp, []byte, err := agent.XML(&xml)
	

````
