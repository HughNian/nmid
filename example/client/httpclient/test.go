package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"nmid-v2/pkg/conf"
)

func main() {
	args := make(map[string]interface{})
	args["name"] = "testtestcontent"
	data, _ := json.Marshal(args)
	req, err := http.NewRequest("POST", "http://127.0.0.1:6809/", bytes.NewReader(data))
	if err != nil {
		log.Fatal("failed to create request: ", err)
		return
	}

	h := req.Header
	h.Set(conf.NRequestType, conf.HTTPDOWORK)
	h.Set(conf.NParamsType, conf.PARAMSTYPEMSGPACK)
	h.Set(conf.NFunctionName, "ToUpper")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("failed to call: ", err)
	}
	defer res.Body.Close()

	// handle http response
	replyData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("failed to read response: ", err)
	}

	log.Println("ret data", string(replyData))
}
