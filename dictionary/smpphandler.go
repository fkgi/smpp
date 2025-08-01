package dictionary

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/fkgi/smpp"
)

var Backend string

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
			smppErr("deliver_sm is notaccepted in Tx BIND", nil)
			return smpp.StatInvCmdID, smpp.MakePDUof(smpp.GenericNack)
		}
	case *smpp.SubmitSM:
		path = "/smppmsg/v1/submit"
		res = &SubmitSM_resp{}
		if info.BindType == smpp.RxBind {
			smppErr("submit_sm is notaccepted in Rx BIND", nil)
			return smpp.StatInvCmdID, smpp.MakePDUof(smpp.GenericNack)
		}
	default:
		smppErr("unknown SMPP request", nil)
		return smpp.StatInvCmdID, smpp.MakePDUof(smpp.GenericNack)
	}

	jsondata, e := json.Marshal(req)
	if e != nil {
		smppErr("failed to marshal request to JSON", e)
		return smpp.StatSysErr, res.unwrap()
	}

	r, e := http.Post(Backend+path, "application/json", bytes.NewBuffer(jsondata))
	if e != nil {
		smppErr("failed to send HTTP request", e)
		return smpp.StatSysErr, res.unwrap()
	}

	if r.StatusCode == http.StatusServiceUnavailable {
		return smpp.StatSysErr, nil
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		return smpp.StatSysErr, smpp.MakePDUof(smpp.GenericNack)
	}

	jsondata, e = io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		smppErr("failed to read HTTP response", e)
		return smpp.StatSysErr, res.unwrap()
	}
	if e = json.Unmarshal(jsondata, res); e != nil {
		smppErr("failed to unmarshal JSON HTTP response", e)
		return smpp.StatSysErr, res.unwrap()
	}

	return res.status(), res.unwrap()
}

func smppErr(s string, e error) {
	if NotifyHandlerError != nil {
		if e != nil {
			NotifyHandlerError("SMPP", s+": "+e.Error())
		} else {
			NotifyHandlerError("SMPP", s)
		}
	}
}
