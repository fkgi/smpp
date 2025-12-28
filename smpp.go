package smpp

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ID = ""
	// WhiteList []BindInfo
	KeepAlive = time.Second * 30
	Expire    = time.Second * 10
	Indent    = "|"
)

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

	closeConnection CommandID = 0xf0000001
	internalFailure CommandID = 0xf0000000
)

func (c CommandID) IsRequest() bool {
	return c&0x80000000 == 0x00000000
}

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

type StatusCode uint32

const (
	StatOK              StatusCode = 0x00000000 // No Error
	StatInvMsgLen       StatusCode = 0x00000001 // Message Length is invalid
	StatInvCmdLen       StatusCode = 0x00000002 // Command Length is invalid
	StatInvCmdID        StatusCode = 0x00000003 // Invalid Command ID
	StatInvBndSts       StatusCode = 0x00000004 // Incorrect BIND Status for given command
	StatAlyBnd          StatusCode = 0x00000005 // ESME Already in Bound State
	StatInvPrtFlg       StatusCode = 0x00000006 // Invalid Priority Flag
	StatInvRegDlvFlg    StatusCode = 0x00000007 // Invalid Registered Delivery Flag
	StatSysErr          StatusCode = 0x00000008 // System Error
	StatInvSrcAdr       StatusCode = 0x0000000A // Invalid Source Address
	StatInvDstAdr       StatusCode = 0x0000000B // Invalid Dest Addr
	StatInvMsgID        StatusCode = 0x0000000C // Message ID is invalid
	StatBindFail        StatusCode = 0x0000000D // Bind Failed
	StatInvPaswd        StatusCode = 0x0000000E // Invalid Password
	StatInvSysID        StatusCode = 0x0000000F // Invalid System ID
	StatCancelFail      StatusCode = 0x00000011 // Cancel SM Failed
	StatReplaceFail     StatusCode = 0x00000013 // Replace SM Failed
	StatMsgQFul         StatusCode = 0x00000014 // Message Queue Full
	StatInvSerTyp       StatusCode = 0x00000015 // Invalid Service Type
	StatInvNumDests     StatusCode = 0x00000033 // Invalid number of destinations
	StatInvDLName       StatusCode = 0x00000034 // Invalid Distribution List name
	StatInvDestFlag     StatusCode = 0x00000040 // Destination flag is invalid(submit_multi)
	StatInvSubRep       StatusCode = 0x00000042 // Invalid ‘submit with replace’ request(i.e. submit_sm with replace_if_present_flag set)
	StatInvEsmClass     StatusCode = 0x00000043 // Invalid esm_class field data
	StatCntSubDL        StatusCode = 0x00000044 // Cannot Submit to Distribution List
	StatSubmitFail      StatusCode = 0x00000045 // submit_sm or submit_multi failed
	StatInvSrcTON       StatusCode = 0x00000048 // Invalid Source address TON
	StatInvSrcNPI       StatusCode = 0x00000049 // Invalid Source address NPI
	StatInvDstTON       StatusCode = 0x00000050 // Invalid Destination address TON
	StatInvDstNPI       StatusCode = 0x00000051 // Invalid Destination address NPI
	StatInvSysTyp       StatusCode = 0x00000053 // Invalid system_type field
	StatInvRepFlag      StatusCode = 0x00000054 // Invalid replace_if_present flag
	StatInvNumMsgs      StatusCode = 0x00000055 // Invalid number of messages
	StatThrottled       StatusCode = 0x00000058 // Throttling error (ESME has exceeded allowed message limits)
	StatInvSched        StatusCode = 0x00000061 // Invalid Scheduled Delivery Time
	StatInvExpiry       StatusCode = 0x00000062 // Invalid message validity period(Expiry time)
	StatInvDFTMsgID     StatusCode = 0x00000063 // Predefined Message Invalid or Not Found
	StatRx_T_AppN       StatusCode = 0x00000064 // ESME Receiver Temporary App Error Code
	StatRx_P_AppN       StatusCode = 0x00000065 // ESME Receiver Permanent App Error Code
	StatRx_R_AppN       StatusCode = 0x00000066 // ESME Receiver Reject Message Error Code
	StatQueryFail       StatusCode = 0x00000067 // query_sm request failed
	StatInvOptParStream StatusCode = 0x000000C0 // Error in the optional part of the PDU Body.
	StatOptParNotAllwd  StatusCode = 0x000000C1 // Optional Parameter not allowed
	StatInvParLen       StatusCode = 0x000000C2 // Invalid Parameter Length.
	StatMissingOptParam StatusCode = 0x000000C3 // Expected Optional Parameter missing
	StatInvOptParamVal  StatusCode = 0x000000C4 // Invalid Optional Parameter Value
	StatDeliveryFailure StatusCode = 0x000000FE // Delivery Failure (used for data_sm_resp)
	StatUnknownErr      StatusCode = 0x000000FF // Unknown Error
)

func (c StatusCode) String() string {
	switch c {
	case StatOK:
		return "ESME_ROK"
	case StatInvMsgLen:
		return "ESME_RINVMSGLEN"
	case StatInvCmdLen:
		return "ESME_RINVCMDLEN"
	case StatInvCmdID:
		return "ESME_RINVCMDID"
	case StatInvBndSts:
		return "ESME_RINVBNDSTS"
	case StatAlyBnd:
		return "ESME_RALYBND"
	case StatInvPrtFlg:
		return "ESME_RINVPRTFLG"
	case StatInvRegDlvFlg:
		return "ESME_RINVREGDLVFLG"
	case StatSysErr:
		return "ESME_RSYSERR"
	case StatInvSrcAdr:
		return "ESME_RINVSRCADR"
	case StatInvDstAdr:
		return "ESME_RINVDSTADR"
	case StatInvMsgID:
		return "ESME_RINVMSGID"
	case StatBindFail:
		return "ESME_RBINDFAIL"
	case StatInvPaswd:
		return "ESME_RINVPASWD"
	case StatInvSysID:
		return "ESME_RINVSYSID"
	case StatCancelFail:
		return "ESME_RCANCELFAIL"
	case StatReplaceFail:
		return "ESME_RREPLACEFAIL"
	case StatMsgQFul:
		return "ESME_RMSGQFUL"
	case StatInvSerTyp:
		return "ESME_RINVSERTYP"
	case StatInvNumDests:
		return "ESME_RINVNUMDESTS"
	case StatInvDLName:
		return "ESME_RINVDLNAME"
	case StatInvDestFlag:
		return "ESME_RINVDESTFLAG"
	case StatInvSubRep:
		return "ESME_RINVSUBREP"
	case StatInvEsmClass:
		return "ESME_RINVESMCLASS"
	case StatCntSubDL:
		return "ESME_RCNTSUBDL"
	case StatSubmitFail:
		return "ESME_RSUBMITFAIL"
	case StatInvSrcTON:
		return "ESME_RINVSRCTON"
	case StatInvSrcNPI:
		return "ESME_RINVSRCNPI"
	case StatInvDstTON:
		return "ESME_RINVDSTTON"
	case StatInvDstNPI:
		return "ESME_RINVDSTNPI"
	case StatInvSysTyp:
		return "ESME_RINVSYSTYP"
	case StatInvRepFlag:
		return "ESME_RINVREPFLAG"
	case StatInvNumMsgs:
		return "ESME_RINVNUMMSGS"
	case StatThrottled:
		return "ESME_RTHROTTLED"
	case StatInvSched:
		return "ESME_RINVSCHED"
	case StatInvExpiry:
		return "ESME_RINVEXPIRY"
	case StatInvDFTMsgID:
		return "ESME_RINVDFTMSGID"
	case StatRx_T_AppN:
		return "ESME_RX_T_APPN"
	case StatRx_P_AppN:
		return "ESME_RX_P_APPN"
	case StatRx_R_AppN:
		return "ESME_RX_R_APPN"
	case StatQueryFail:
		return "ESME_RQUERYFAIL"
	case StatInvOptParStream:
		return "ESME_RINVOPTPARSTREAM"
	case StatOptParNotAllwd:
		return "ESME_ROPTPARNOTALLWD"
	case StatInvParLen:
		return "ESME_RINVPARLEN"
	case StatMissingOptParam:
		return "ESME_RMISSINGOPTPARAM"
	case StatInvOptParamVal:
		return "ESME_RINVOPTPARAMVAL"
	case StatDeliveryFailure:
		return "ESME_RDELIVERYFAILURE"
	case StatUnknownErr:
		return "ESME_RUNKNOWNERR"
	}
	return "Reserved"
}

func (c StatusCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *StatusCode) UnmarshalJSON(b []byte) (e error) {
	s := ""
	if e = json.Unmarshal(b, &s); e != nil {
		return
	}
	switch s {
	case "ESME_ROK":
		*c = StatOK
	case "ESME_RINVMSGLEN":
		*c = StatInvMsgLen
	case "ESME_RINVCMDLEN":
		*c = StatInvCmdLen
	case "ESME_RINVCMDID":
		*c = StatInvCmdID
	case "ESME_RINVBNDSTS":
		*c = StatInvBndSts
	case "ESME_RALYBND":
		*c = StatAlyBnd
	case "ESME_RINVPRTFLG":
		*c = StatInvPrtFlg
	case "ESME_RINVREGDLVFLG":
		*c = StatInvRegDlvFlg
	case "ESME_RSYSERR":
		*c = StatSysErr
	case "ESME_RINVSRCADR":
		*c = StatInvSrcAdr
	case "ESME_RINVDSTADR":
		*c = StatInvDstAdr
	case "ESME_RINVMSGID":
		*c = StatInvMsgID
	case "ESME_RBINDFAIL":
		*c = StatBindFail
	case "ESME_RINVPASWD":
		*c = StatInvPaswd
	case "ESME_RINVSYSID":
		*c = StatInvSysID
	case "ESME_RCANCELFAIL":
		*c = StatCancelFail
	case "ESME_RREPLACEFAIL":
		*c = StatReplaceFail
	case "ESME_RMSGQFUL":
		*c = StatMsgQFul
	case "ESME_RINVSERTYP":
		*c = StatInvSerTyp
	case "ESME_RINVNUMDESTS":
		*c = StatInvNumDests
	case "ESME_RINVDLNAME":
		*c = StatInvDLName
	case "ESME_RINVDESTFLAG":
		*c = StatInvDestFlag
	case "ESME_RINVSUBREP":
		*c = StatInvSubRep
	case "ESME_RINVESMCLASS":
		*c = StatInvEsmClass
	case "ESME_RCNTSUBDL":
		*c = StatCntSubDL
	case "ESME_RSUBMITFAIL":
		*c = StatSubmitFail
	case "ESME_RINVSRCTON":
		*c = StatInvSrcTON
	case "ESME_RINVSRCNPI":
		*c = StatInvSrcNPI
	case "ESME_RINVDSTTON":
		*c = StatInvDstTON
	case "ESME_RINVDSTNPI":
		*c = StatInvDstNPI
	case "ESME_RINVSYSTYP":
		*c = StatInvSysTyp
	case "ESME_RINVREPFLAG":
		*c = StatInvRepFlag
	case "ESME_RINVNUMMSGS":
		*c = StatInvNumMsgs
	case "ESME_RTHROTTLED":
		*c = StatThrottled
	case "ESME_RINVSCHED":
		*c = StatInvSched
	case "ESME_RINVEXPIRY":
		*c = StatInvExpiry
	case "ESME_RINVDFTMSGID":
		*c = StatInvDFTMsgID
	case "ESME_RX_T_APPN":
		*c = StatRx_T_AppN
	case "ESME_RX_P_APPN":
		*c = StatRx_P_AppN
	case "ESME_RX_R_APPN":
		*c = StatRx_R_AppN
	case "ESME_RQUERYFAIL":
		*c = StatQueryFail
	case "ESME_RINVOPTPARSTREAM":
		*c = StatInvOptParStream
	case "ESME_ROPTPARNOTALLWD":
		*c = StatOptParNotAllwd
	case "ESME_RINVPARLEN":
		*c = StatInvParLen
	case "ESME_RMISSINGOPTPARAM":
		*c = StatMissingOptParam
	case "ESME_RINVOPTPARAMVAL":
		*c = StatInvOptParamVal
	case "ESME_RDELIVERYFAILURE":
		*c = StatDeliveryFailure
	case "ESME_RUNKNOWNERR":
		*c = StatUnknownErr
	default:
		e = errors.New("undefined code")
	}
	return
}
