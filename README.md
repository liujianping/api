# api
===

Simplified HTTP client for API wrapper

##  Quick Start

````go

	import "github.com/liujianping/api"

	//! create api request agent

	agent := api.URL("http://a.domain.com/")
	agent := api.HTTP("host:port", port)
	agent := api.HTTPs("host", port)
	
	agent := api.Get("http://a.domain.com/")
	// agent := api.URL("http://a.domain.com/").Method(api.GET)

	agent := api.Post("http://a.domain.com/")
	agent := api.Patch("http://a.domain.com/")
	agent := api.Put("http://a.domain.com/")
	agent := api.Head("http://a.domain.com/")

	//! set api request method & headers & parameters & form-data

	agent.HeadSet("key", "value")
	agent.HeadDel("key", "value")

	agent.QuerySet("key", "value")
	agent.QueryAdd("key", "value")
	agent.QueryDel("key", "value")

	agent.FormData(form)
	agent.JSONData(obj)
	agent.XMLData(obj)

	//! do api request

	resp, []byte, err := agent.Bytes()

	resp, string, err := agent.Text()

	resp, []byte, err := agent.JSON(&json)

	resp, []byte, err := agent.XML(&xml)
	

````
