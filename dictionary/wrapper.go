package dictionary

import "github.com/fkgi/smpp"

type wrappedResp interface {
	status() smpp.StatusCode
	unwrap() smpp.PDU
}

type DataSM_resp struct {
	Status smpp.StatusCode `json:"command_status"`
	smpp.DataSM_resp
}

func (r *DataSM_resp) status() smpp.StatusCode { return r.Status }
func (r *DataSM_resp) unwrap() smpp.PDU        { return &r.DataSM_resp }

type DeliverSM_resp struct {
	Status smpp.StatusCode `json:"command_status"`
	smpp.DeliverSM_resp
}

func (r *DeliverSM_resp) status() smpp.StatusCode { return r.Status }
func (r *DeliverSM_resp) unwrap() smpp.PDU        { return &r.DeliverSM_resp }

type SubmitSM_resp struct {
	Status smpp.StatusCode `json:"command_status"`
	smpp.SubmitSM_resp
}

func (r *SubmitSM_resp) status() smpp.StatusCode { return r.Status }
func (r *SubmitSM_resp) unwrap() smpp.PDU        { return &r.SubmitSM_resp }

type GenericNack struct {
	Status smpp.StatusCode `json:"command_status"`
}

func (r *GenericNack) status() smpp.StatusCode { return r.Status }
func (r *GenericNack) unwrap() smpp.PDU        { return nil }

var NotifyHandlerError func(proto, msg string) = nil
