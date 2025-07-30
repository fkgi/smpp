package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/fkgi/smpp"
)

func handleSMPP(info smpp.BindInfo, req smpp.PDU) (smpp.StatusCode, smpp.PDU) {
	var res wrappedResp
	var path string
	switch req.(type) {
	case *smpp.DataSM:
		path = "/smppmsg/v1/data"
		res = &DataSM_resp{}
	case *smpp.DeliverSM:
		path = "/smppmsg/v1/deliver"
		res = &DeliverSM_resp{}
	case *smpp.SubmitSM:
		path = "/smppmsg/v1/submit"
		res = &SubmitSM_resp{}
	default:
		log.Println("[ERROR]", "unknown SMPP request")
		return smpp.StatSysErr, smpp.MakePDUof(smpp.GenericNack)
	}

	jsondata, e := json.Marshal(req)
	if e != nil {
		log.Println("[ERROR]", "failed to marshal request to JSON:", e)
		return smpp.StatSysErr, res.unwrap()
	}

	r, e := http.Post(backend+path, "application/json", bytes.NewBuffer(jsondata))
	if e != nil {
		log.Println("[ERROR]", "failed to send HTTP request:", e)
		return smpp.StatSysErr, res.unwrap()
	}

	if r.StatusCode == http.StatusServiceUnavailable {
		log.Println("[INFO]", "reject request")
		return smpp.StatSysErr, nil
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		log.Println("[INFO]", "error response from HTTP")
		return smpp.StatSysErr, smpp.MakePDUof(smpp.GenericNack)
	}

	jsondata, e = io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("[ERROR]", "failed to read HTTP response:", e)
		return smpp.StatSysErr, res.unwrap()
	}
	if e = json.Unmarshal(jsondata, res); e != nil {
		log.Println("[ERROR]", "failed to unmarshal JSON HTTP response:", e)
		return smpp.StatSysErr, res.unwrap()
	}

	return res.status(), res.unwrap()
}
