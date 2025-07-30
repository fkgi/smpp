package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/fkgi/smpp"
)

func handleHTTP(w http.ResponseWriter, r *http.Request, req smpp.PDU, b smpp.Bind) {
	if r.Method != http.MethodPost {
		log.Println("[INFO]", "invalid HTTP request method:", r.Method)
		w.Header().Add("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	jsondata, e := io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("[ERROR]", "failed to read request:", e)
		httpErr("unable to read request body", e.Error(), http.StatusBadRequest, w)
		return
	}
	if e = json.Unmarshal(jsondata, &req); e != nil {
		log.Println("[ERROR]", "failed to unmarshal request from JSON:", e)
		httpErr("unexpected JSON data", e.Error(), http.StatusBadRequest, w)
		return
	}

	stat, res, e := b.Send(req)
	if e != nil {
		log.Println("[ERROR]", "failed to send SMPP request:", e)
		httpErr("failed to send request", e.Error(), http.StatusInternalServerError, w)
		return
	}

	switch res := res.(type) {
	case *smpp.DataSM_resp:
		jsondata, e = json.Marshal(&DataSM_resp{
			Status:      stat,
			DataSM_resp: *res})
	case *smpp.DeliverSM_resp:
		jsondata, e = json.Marshal(&DeliverSM_resp{
			Status:         stat,
			DeliverSM_resp: *res})
	case *smpp.SubmitSM_resp:
		jsondata, e = json.Marshal(&SubmitSM_resp{
			Status:        stat,
			SubmitSM_resp: *res})
	default:
		switch res.CommandID() {
		case smpp.GenericNack:
			jsondata, e = json.Marshal(&GenericNack{
				Status: stat})
		default:
			e = errors.New("unknown SMPP request")
		}
	}

	if e != nil {
		log.Println("[ERROR]", "failed to marshal response to JSON:", e)
		httpErr("failed to unmarshal response", e.Error(), http.StatusInternalServerError, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsondata)
}

func httpErr(title, detail string, code int, w http.ResponseWriter) {
	data, _ := json.Marshal(struct {
		T string `json:"title"`
		D string `json:"detail"`
	}{T: title, D: detail})

	w.Header().Add("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	w.Write(data)
}
