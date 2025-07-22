package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/fkgi/smpp"
)

func handleSMPP(info smpp.BindInfo, req smpp.Request) (res smpp.Response) {
	var path string
	switch req.(type) {
	case *smpp.DataSM:
		path = "/smppmsg/v1/data"
		res = &smpp.DataSM_resp{Status: smpp.StatSysErr}
	case *smpp.DeliverSM:
		path = "/smppmsg/v1/deliver"
		res = &smpp.DeliverSM_resp{Status: smpp.StatSysErr}
	case *smpp.SubmitSM:
		path = "/smppmsg/v1/submit"
		res = &smpp.SubmitSM_resp{Status: smpp.StatSysErr}
	default:
		log.Println("[ERROR]", "unknown SMPP request")
		return
	}

	jsondata, e := json.Marshal(req)
	if e != nil {
		log.Println("[ERROR]", "failed to marshal request to JSON:", e)
		return
	}

	r, e := http.Post(backend+path, "application/json", bytes.NewBuffer(jsondata))
	if e != nil {
		log.Println("[ERROR]", "failed to send HTTP request:", e)
		return
	}

	jsondata, e = io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("[ERROR]", "failed to read HTTP response:", e)
	} else if e = json.Unmarshal(jsondata, res); e != nil {
		log.Println("[ERROR]", "failed to unmarshal JSON HTTP response:", e)
	}
	return
}
