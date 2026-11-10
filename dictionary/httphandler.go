package dictionary

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/fkgi/smpp"
)

func HandleHTTP(w http.ResponseWriter, r *http.Request, req smpp.PDU, b *smpp.Bind) {
	txJson, e := io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		httpErr(r.URL.Path, nil, http.StatusBadRequest,
			"unable to read HTTP request body", e.Error(), w)
		return
	}
	if e = json.Unmarshal(txJson, &req); e != nil {
		httpErr(r.URL.Path, txJson, http.StatusBadRequest,
			"invalid JSON data", e.Error(), w)
		return
	}

	stat, res, e := b.Send(req)
	if e != nil {
		httpErr(r.URL.Path, txJson, http.StatusInternalServerError,
			"failed to send SMPP request", e.Error(), w)
		return
	}

	var rxJson []byte
	switch res := res.(type) {
	case *smpp.DataSM_resp:
		rxJson, e = json.Marshal(&DataSM_resp{
			Status:      stat,
			DataSM_resp: *res})
	case *smpp.DeliverSM_resp:
		rxJson, e = json.Marshal(&DeliverSM_resp{
			Status:         stat,
			DeliverSM_resp: *res})
	case *smpp.SubmitSM_resp:
		rxJson, e = json.Marshal(&SubmitSM_resp{
			Status:        stat,
			SubmitSM_resp: *res})
	default:
		switch res.CommandID() {
		case smpp.GenericNack:
			rxJson, e = json.Marshal(&GenericNack{Status: stat})
		default:
			e = errors.New("unknown SMPP request")
		}
	}

	if e != nil {
		httpErr(r.URL.Path, txJson, http.StatusInternalServerError,
			"unable to unmarshal response to JSON", e.Error(), w)
		return
	}

	if TraceRxHttpRequest != nil {
		TraceRxHttpRequest(r.URL.Path, txJson, http.StatusOK, rxJson, nil)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rxJson)
}

var TraceRxHttpRequest func(string, []byte, int, []byte, error) = nil

func httpErr(
	path string, txj []byte, hcode int,
	title, detail string, w http.ResponseWriter) {
	if NotifyHandlerError != nil {
		NotifyHandlerError("HTTP", title+": "+detail)
	}

	data, _ := json.Marshal(struct {
		T string `json:"title"`
		D string `json:"detail"`
	}{T: title, D: detail})

	if TraceRxHttpRequest != nil {
		TraceRxHttpRequest(path, txj, hcode, data, errors.New(title+": "+detail))
	}

	w.Header().Add("Content-Type", "application/problem+json")
	w.WriteHeader(hcode)
	w.Write(data)
}
