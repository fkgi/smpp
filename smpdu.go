package smpp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fkgi/teldata"
)

type smPDU struct {
	SvcType  string                  `json:"svc_type,omitempty"`
	SrcTON   teldata.NatureOfAddress `json:"src_ton,omitempty"`
	SrcNPI   teldata.NumberingPlan   `json:"src_npi,omitempty"`
	SrcAddr  string                  `json:"src_addr,omitempty"`
	DstTON   teldata.NatureOfAddress `json:"dst_ton"`
	DstNPI   teldata.NumberingPlan   `json:"dst_npi"`
	DstAddr  string                  `json:"dst_addr"`
	EsmClass esmClass                `json:"esm_class"`

	ProtocolId           byte               `json:"protocol_id"`
	PriorityFlag         byte               `json:"priority_flag"`
	ScheduleDeliveryTime string             `json:"schedule_delivery_time,omitempty"`
	ValidityPeriod       string             `json:"validity_period,omitempty"`
	RegisteredDelivery   registeredDelivery `json:"registered_delivery"`
	ReplaceIfPresentFlag bool               `json:"replace_if_present_flag,omitempty"`
	DataCoding           byte               `json:"data_coding"`
	SmDefaultMsgId       byte               `json:"sm_default_sm_id,omitempty"`
	// SmLength            byte
	ShortMessage UserData `json:"short_message,omitempty"`

	Param OptionalParameters `json:"options,omitempty"`
}

func (d *smPDU) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, "service_type           :", d.SvcType)
	fmt.Fprintln(buf, Indent, "source_addr_ton        :", d.SrcTON)
	fmt.Fprintln(buf, Indent, "source_addr_npi        :", d.SrcNPI)
	fmt.Fprintln(buf, Indent, "source_addr            :", d.SrcAddr)
	fmt.Fprintln(buf, Indent, "dest_addr_ton          :", d.DstTON)
	fmt.Fprintln(buf, Indent, "dest_addr_npi          :", d.DstNPI)
	fmt.Fprintln(buf, Indent, "destination_addr       :", d.DstAddr)
	fmt.Fprintln(buf, Indent, "esm_class              :", d.EsmClass)
	fmt.Fprintln(buf, Indent, "protocol_id            :", d.ProtocolId)
	fmt.Fprintln(buf, Indent, "priority_flag          :", d.PriorityFlag)
	fmt.Fprintln(buf, Indent, "schedule_delivery_time :", d.ScheduleDeliveryTime)
	fmt.Fprintln(buf, Indent, "validity_period        :", d.ValidityPeriod)
	fmt.Fprintln(buf, Indent, "registered_delivery    :", d.RegisteredDelivery)
	fmt.Fprintln(buf, Indent, "replace_if_present_flag:", d.ReplaceIfPresentFlag)
	fmt.Fprintln(buf, Indent, "data_coding            :", d.DataCoding)
	fmt.Fprintln(buf, Indent, "sm_default_msg_id      :", d.SmDefaultMsgId)
	// fmt.Fprintln(buf, Indent, "sm_length              :", len(d.ShortMessage))
	fmt.Fprintln(buf, Indent, "short_message          :", d.ShortMessage)
	fmt.Fprint(buf, d.Param)
	return buf.String()
}

func (d *smPDU) Marshal(v byte) []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.SvcType), w)
	writeAddr(d.SrcTON, d.SrcNPI, d.SrcAddr, w)
	writeAddr(d.DstTON, d.DstNPI, d.DstAddr, w)
	if len(d.ShortMessage.UDH) != 0 {
		d.EsmClass.UDHI = true
	}
	d.EsmClass.writeTo(w)
	w.WriteByte(d.ProtocolId)
	w.WriteByte(d.PriorityFlag)
	writeCString([]byte(d.ScheduleDeliveryTime), w)
	writeCString([]byte(d.ValidityPeriod), w)
	d.RegisteredDelivery.writeTo(w)
	writeBool(d.ReplaceIfPresentFlag, w)
	w.WriteByte(d.DataCoding)
	w.WriteByte(d.SmDefaultMsgId)

	ud := d.ShortMessage.marshal(d.DataCoding)
	w.WriteByte(byte(len(ud)))
	w.Write(ud)

	if v >= 0x34 {
		d.Param.writeTo(w)
	}
	return w.Bytes()
}

func (d *smPDU) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	var l byte
	if d.SvcType, e = readCString(buf); e != nil {
	} else if d.SrcTON, d.SrcNPI, d.SrcAddr, e = readAddr(buf); e != nil {
	} else if d.DstTON, d.DstNPI, d.DstAddr, e = readAddr(buf); e != nil {
	} else if e = d.EsmClass.readFrom(buf); e != nil {
	} else if d.ProtocolId, e = buf.ReadByte(); e != nil {
	} else if d.PriorityFlag, e = buf.ReadByte(); e != nil {
	} else if d.ScheduleDeliveryTime, e = readCString(buf); e != nil {
	} else if d.ValidityPeriod, e = readCString(buf); e != nil {
	} else if e = d.RegisteredDelivery.readFrom(buf); e != nil {
	} else if d.ReplaceIfPresentFlag, e = readBool(buf); e != nil {
	} else if d.DataCoding, e = buf.ReadByte(); e != nil {
	} else if d.SmDefaultMsgId, e = buf.ReadByte(); e != nil {
	} else if l, e = buf.ReadByte(); e == nil {
		ud := make([]byte, int(l))
		if _, e = buf.Read(ud); e != nil {
		} else if e = d.ShortMessage.unmarshal(ud, d.DataCoding, d.EsmClass.UDHI); e != nil {
		} else {
			d.Param = OptionalParameters{}
			e = d.Param.readFrom(buf)
		}
	}
	return
}

type messagingMode byte

const (
	DefaultSMSC     messagingMode = 0x00
	Datagram        messagingMode = 0x01
	Forward         messagingMode = 0x02
	StoreAndForward messagingMode = 0x03
)

func (m messagingMode) String() string {
	switch m {
	case DefaultSMSC:
		return "default_SMSC"
	case Datagram:
		return "datagram"
	case Forward:
		return "forward"
	case StoreAndForward:
		return "store_and_forward"
	}
	return "unknown"
}

func (m messagingMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *messagingMode) UnmarshalJSON(b []byte) (e error) {
	s := ""
	if e = json.Unmarshal(b, &s); e != nil {
		return
	}
	switch s {
	case "default_SMSC":
		*m = DefaultSMSC
	case "datagram":
		*m = Datagram
	case "forward":
		*m = Forward
	case "store_andF_frward":
		*m = StoreAndForward
	default:
		e = errors.New("invalid Messaging Mode: " + s)
	}
	return
}

type messageType byte

const (
	DefaultMsg         messageType = 0x00
	DeliveryReceipt    messageType = 0x04
	DeliveryAck        messageType = 0x08
	ManualUserAck      messageType = 0x10
	ConversationAbort  messageType = 0x18
	InterDeliveryNotif messageType = 0x20
)

func (m messageType) String() string {
	switch m {
	case DefaultMsg:
		return "default_msg"
	case DeliveryReceipt:
		return "delivery_receipt"
	case DeliveryAck:
		return "delivery_ack"
	case ManualUserAck:
		return "manual/user_ack"
	case ConversationAbort:
		return "conversation_abort"
	case InterDeliveryNotif:
		return "intermadiate_delivery_notification"
	}
	return "unknown"
}

func (m messageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *messageType) UnmarshalJSON(b []byte) (e error) {
	s := ""
	if e = json.Unmarshal(b, &s); e != nil {
		return
	}
	switch s {
	case "default_msg":
		*m = DefaultMsg
	case "delivery_receipt":
		*m = DeliveryReceipt
	case "delivery_ack":
		*m = DeliveryAck
	case "manual/user_ack":
		*m = ManualUserAck
	case "conversation_abort":
		*m = ConversationAbort
	case "intermadiate_delivery_notification":
		*m = InterDeliveryNotif
	default:
		e = errors.New("invalid Messaging Type")
	}
	return
}

type esmClass struct {
	Mode      messagingMode `json:"message_mode"`
	Type      messageType   `json:"message_type"`
	UDHI      bool          `json:"udhi_indicator"`
	ReplyPath bool          `json:"reply_path"`
}

func (c esmClass) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, Indent, "message_mode   :", c.Mode)
	fmt.Fprintln(buf, Indent, Indent, "message_type   :", c.Type)
	fmt.Fprintln(buf, Indent, Indent, "udhi_indicator :", c.UDHI)
	fmt.Fprint(buf, Indent, " ", Indent, " reply_path     : ", c.ReplyPath)
	return buf.String()
}

func (c esmClass) writeTo(buf *bytes.Buffer) {
	b := byte(c.Mode) | byte(c.Type)
	if c.UDHI {
		b |= 0x40
	}
	if c.ReplyPath {
		b |= 0x80
	}
	buf.WriteByte(b)
}

func (c *esmClass) readFrom(buf *bytes.Buffer) error {
	b, e := buf.ReadByte()
	if e == nil {
		c.Mode = messagingMode(b & 0x03)
		c.Type = messageType(b & 0x3c)
		c.UDHI = b&0x40 == 0x40
		c.ReplyPath = b&0x80 == 0x80
	}
	return e
}

type deliveryReceipt byte

const (
	NoReceipt      deliveryReceipt = 0x00
	ReceiptOnAll   deliveryReceipt = 0x01
	ReceiptOnError deliveryReceipt = 0x02
)

func (r deliveryReceipt) String() string {
	switch r {
	case NoReceipt:
		return "no_delivery_receipt_requested"
	case ReceiptOnAll:
		return "delivery_receipt_requested_on_success_or_failure"
	case ReceiptOnError:
		return "delivery_receipt_requested_on_failure"
	}
	return "unknown"
}

func (r deliveryReceipt) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *deliveryReceipt) UnmarshalJSON(b []byte) (e error) {
	s := ""
	if e = json.Unmarshal(b, &s); e != nil {
		return
	}
	switch s {
	case "no_delivery_receipt_requested":
		*r = NoReceipt
	case "delivery_receipt_requested_on_success_or_failure":
		*r = ReceiptOnAll
	case "delivery_receipt_requested_on_failure":
		*r = ReceiptOnError
	default:
		e = errors.New("invalid Delivery Receipt")
	}
	return
}

type registeredDelivery struct {
	Receipt           deliveryReceipt `json:"delivery_receipt"`
	DeliveryAck       bool            `json:"delivery_ack"`
	ManualUserAck     bool            `json:"manual/user_ack"`
	IntermediateNotif bool            `json:"intermadiate_delivery_notification"`
}

func (r registeredDelivery) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, Indent, "delivery_receipt                  :", r.Receipt)
	fmt.Fprintln(buf, Indent, Indent, "delivery_ack                      :", r.DeliveryAck)
	fmt.Fprintln(buf, Indent, Indent, "manual/user_ack                   :", r.ManualUserAck)
	fmt.Fprint(buf, Indent, " ", Indent, " intermadiate_delivery_notification: ", r.IntermediateNotif)
	return buf.String()
}

func (r registeredDelivery) writeTo(buf *bytes.Buffer) {
	b := byte(r.Receipt)
	if r.DeliveryAck {
		b |= 0x04
	}
	if r.ManualUserAck {
		b |= 0x08
	}
	if r.IntermediateNotif {
		b |= 0x10
	}
	buf.WriteByte(b)
}

func (r *registeredDelivery) readFrom(buf *bytes.Buffer) error {
	b, e := buf.ReadByte()
	if e == nil {
		r.Receipt = deliveryReceipt(b & 0x03)
		r.DeliveryAck = b&0x04 == 0x04
		r.ManualUserAck = b&0x08 == 0x08
		r.IntermediateNotif = b&0x10 == 0x10
	}
	return e
}

type SubmitSM struct {
	smPDU
}

func (*SubmitSM) CommandID() CommandID { return SubmitSm }

type SubmitSM_resp struct {
	MessageID string `json:"id,omitempty"`
}

func (d *SubmitSM_resp) String() string {
	return fmt.Sprint("\n", Indent, " id: ", d.MessageID)
}
func (*SubmitSM_resp) CommandID() CommandID { return SubmitSmResp }

func (d *SubmitSM_resp) Marshal(byte) []byte {
	w := new(bytes.Buffer)
	if len(d.MessageID) != 0 {
		writeCString([]byte(d.MessageID), w)
	}
	return w.Bytes()
}

func (d *SubmitSM_resp) Unmarshal(data []byte) (e error) {
	if len(data) != 0 {
		buf := bytes.NewBuffer(data)
		d.MessageID, e = readCString(buf)
	}
	return
}

type DeliverSM struct {
	smPDU
}

func (*DeliverSM) CommandID() CommandID { return DeliverSm }

type DeliverSM_resp struct{}

func (d *DeliverSM_resp) String() string     { return "" }
func (*DeliverSM_resp) CommandID() CommandID { return DeliverSmResp }

func (d *DeliverSM_resp) Marshal(byte) []byte {
	w := bytes.Buffer{}
	writeCString([]byte{}, &w)
	return w.Bytes()
}

func (d *DeliverSM_resp) Unmarshal(data []byte) (e error) {
	return
}
