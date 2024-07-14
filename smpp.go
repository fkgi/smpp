package smpp

import (
	"time"
)

var (
	ID        = ""
	WhiteList []BindInfo
	KeepAlive = time.Second * 30
	Expire    = time.Second

	sequence = make(chan uint32, 1)
)

func init() {
	sequence <- 1
}

func nextSequence() uint32 {
	ret := <-sequence
	if ret == 0x7fffffff {
		sequence <- 1
	} else {
		sequence <- ret + 1
	}
	return ret
}

type CommandID uint32

const (
	GenericNack         CommandID = 0x80000000
	BindReceiver        CommandID = 0x00000001
	BindReceiverResp    CommandID = 0x80000001
	BindTransmitter     CommandID = 0x00000002
	BindTransmitterResp CommandID = 0x80000002
	QuerySm             CommandID = 0x00000003
	QuerySmResp         CommandID = 0x80000003
	SubmitSm            CommandID = 0x00000004
	SubmitSmResp        CommandID = 0x80000004
	DeliverSm           CommandID = 0x00000005
	DeliverSmResp       CommandID = 0x80000005
	Unbind              CommandID = 0x00000006
	UnbindResp          CommandID = 0x80000006
	ReplaceSm           CommandID = 0x00000007
	ReplaceSmResp       CommandID = 0x80000007
	CancelSm            CommandID = 0x00000008
	CancelSmResp        CommandID = 0x80000008
	BindTransceiver     CommandID = 0x00000009
	BindTransceiverResp CommandID = 0x80000009
	Outbind             CommandID = 0x0000000B
	EnquireLink         CommandID = 0x00000015
	EnquireLinkResp     CommandID = 0x80000015
	SubmitMulti         CommandID = 0x00000021
	SubmitMultiResp     CommandID = 0x80000021
	AlertNotification   CommandID = 0x00000102
	DataSm              CommandID = 0x00000103
	DataSmResp          CommandID = 0x80000103

	CloseConnection CommandID = 0xf0000001
	InternalFailure CommandID = 0xf0000000
)

func (c CommandID) String() string {
	switch c {
	case GenericNack:
		return "generic_nack"
	case BindReceiver:
		return "bind_receiver"
	case BindReceiverResp:
		return "bind_receiver_resp"
	case BindTransmitter:
		return "bind_transmitter"
	case BindTransmitterResp:
		return "bind_transmitter_resp"
	case QuerySm:
		return "query_sm"
	case QuerySmResp:
		return "query_sm_resp"
	case SubmitSm:
		return "submit_sm"
	case SubmitSmResp:
		return "submit_sm_resp"
	case DeliverSm:
		return "deliver_sm"
	case DeliverSmResp:
		return "deliver_sm_resp"
	case Unbind:
		return "unbind"
	case UnbindResp:
		return "unbind_resp"
	case ReplaceSm:
		return "replace_sm"
	case ReplaceSmResp:
		return "replace_sm_resp"
	case CancelSm:
		return "cancel_sm"
	case CancelSmResp:
		return "cancel_sm_resp"
	case BindTransceiver:
		return "bind_transceiver"
	case BindTransceiverResp:
		return "bind_transceiver_resp"
	case Outbind:
		return "outbind"
	case EnquireLink:
		return "enquire_link"
	case EnquireLinkResp:
		return "enquire_link_resp"
	case SubmitMulti:
		return "submit_multi"
	case SubmitMultiResp:
		return "submit_multi_resp"
	case AlertNotification:
		return "alert_notification"
	case DataSm:
		return "data_sm"
	case DataSmResp:
		return "data_sm_resp"
	}
	return "reserved"
}
