package dictionary

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/fkgi/smpp"
)

var Backend string
var Client http.Client

func init() {
	t, _ := http.DefaultTransport.(*http.Transport)
	dt := t.Clone()
	dt.MaxIdleConns = 0
	dt.MaxIdleConnsPerHost = 1000
	Client = http.Client{Transport: dt, Timeout: smpp.Expire}
}

func HandleSMPP(info smpp.BindInfo, req smpp.PDU) (smpp.StatusCode, smpp.PDU) {
	var res wrappedResp
	var path string
	switch req.(type) {
	case *smpp.DataSM:
		path = "/smppmsg/v1/data"
		res = &DataSM_resp{}
	case *smpp.DeliverSM:
		path = "/smppmsg/v1/deliver"
		res = &DeliverSM_resp{}
		if info.BindType == smpp.TxBind {
			if TraceTxHttpRequest != nil {
				TraceTxHttpRequest(path, nil, 0, nil,
					errors.New("deliver_sm is notaccepted in Tx BIND"))
			}
			return smpp.StatInvCmdID, smpp.MakePDUof(smpp.GenericNack)
		}
	case *smpp.SubmitSM:
		path = "/smppmsg/v1/submit"
		res = &SubmitSM_resp{}
		if info.BindType == smpp.RxBind {
			if TraceTxHttpRequest != nil {
				TraceTxHttpRequest(path, nil, 0, nil,
					errors.New("submit_sm is notaccepted in Rx BIND"))
			}
			return smpp.StatInvCmdID, smpp.MakePDUof(smpp.GenericNack)
		}
	default:
		if TraceTxHttpRequest != nil {
			TraceTxHttpRequest(path, nil, 0, nil,
				errors.New("unknown SMPP request"))
		}
		return smpp.StatInvCmdID, smpp.MakePDUof(smpp.GenericNack)
	}

	txJson, e := json.Marshal(req)
	if e != nil {
		if TraceTxHttpRequest != nil {
			e = errors.New("failed to marshal request to JSON: " + e.Error())
			TraceTxHttpRequest(path, nil, 0, nil, e)
		}
		return smpp.StatSysErr, res.unwrap()
	}

	r, e := Client.Post(Backend+path, "application/json", bytes.NewBuffer(txJson))
	if e != nil {
		if TraceTxHttpRequest != nil {
			e = errors.New("unable to send HTTP request to backend: " + e.Error())
			TraceTxHttpRequest(path, txJson, 0, nil, e)
		}
		return smpp.StatSysErr, res.unwrap()
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case http.StatusServiceUnavailable:
		if TraceTxHttpRequest != nil {
			TraceTxHttpRequest(path, txJson, r.StatusCode, nil, nil)
		}
		return smpp.StatSysErr, nil
	case http.StatusOK, http.StatusCreated:
	default:
		if TraceTxHttpRequest != nil {
			e = errors.New("unknown error in HTTP")
			TraceTxHttpRequest(path, txJson, r.StatusCode, nil, e)
		}
		return smpp.StatSysErr, smpp.MakePDUof(smpp.GenericNack)
	}

	rxJson, e := io.ReadAll(r.Body)
	if e != nil {
		if TraceTxHttpRequest != nil {
			e = errors.New("failed to read HTTP response: " + e.Error())
			TraceTxHttpRequest(path, txJson, r.StatusCode, nil, e)
		}
		return smpp.StatSysErr, res.unwrap()
	}

	if e = json.Unmarshal(rxJson, res); e != nil {
		if TraceTxHttpRequest != nil {
			e = errors.New("failed to unmarshal JSON HTTP response: " + e.Error())
			TraceTxHttpRequest(path, txJson, r.StatusCode, rxJson, e)
		}
		return smpp.StatSysErr, res.unwrap()
	}

	if TraceTxHttpRequest != nil {
		TraceTxHttpRequest(path, txJson, r.StatusCode, rxJson, nil)
	}
	return res.status(), res.unwrap()
}

var TraceTxHttpRequest func(string, []byte, int, []byte, error) = nil
