package smpp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fkgi/teldata"
)

type PDU interface {
	CommandID() CommandID
	Marshal(byte) []byte
	Unmarshal([]byte) error
	fmt.Stringer
}

func MakePDUof(c CommandID) PDU {
	switch c {
	case GenericNack:
		return &genericNack{}
	case BindReceiver, BindTransmitter, BindTransceiver:
		return &bindReq{cmd: c}
	case BindReceiverResp, BindTransmitterResp, BindTransceiverResp:
		return &bindRes{cmd: c}
	// case QuerySm:
	// case QuerySmResp:
	case SubmitSm:
		return &SubmitSM{}
	case SubmitSmResp:
		return &SubmitSM_resp{}
	case DeliverSm:
		return &DeliverSM{}
	case DeliverSmResp:
		return &DeliverSM_resp{}
	case Unbind:
		return &unbindReq{}
	case UnbindResp:
		return &unbindRes{}
	//case ReplaceSm:
	//case ReplaceSmResp:
	//case CancelSm:
	//case CancelSmResp:
	//case Outbind:
	case EnquireLink:
		return &enquireReq{}
	case EnquireLinkResp:
		return &enquireRes{}
	// case SubmitMulti:
	// case SubmitMultiResp:
	// case AlertNotification:
	case DataSm:
		return &DataSM{}
	case DataSmResp:
		return &DataSM_resp{}
	}
	return nil
}

func (b *Bind) readPDU() (msg message, e error) {
	var l uint32
	if e = binary.Read(b.con, binary.BigEndian, &l); e != nil {
		return
	}
	if l < 16 {
		e = errors.New("invalid header")
		return
	}
	l -= 16
	if e = binary.Read(b.con, binary.BigEndian, &(msg.id)); e != nil {
		return
	}
	if e = binary.Read(b.con, binary.BigEndian, &(msg.stat)); e != nil {
		return
	}
	if e = binary.Read(b.con, binary.BigEndian, &(msg.seq)); e != nil {
		return
	}
	if l != 0 {
		msg.body = make([]byte, l)
		offset := 0
		n := 0
		for offset < 1 {
			n, e = b.con.Read(msg.body[offset:])
			offset += n
			if e != nil {
				break
			}
		}
	}

	if TraceMessage != nil {
		TraceMessage(Rx, msg.id, msg.stat, msg.seq, msg.body)
	}
	return
}

func (b *Bind) writePDU(msg message) (e error) {
	if msg.body == nil {
		msg.body = []byte{}
	}
	buf := bufio.NewWriter(b.con)

	// command_length
	binary.Write(buf, binary.BigEndian, uint32(len(msg.body)+16))
	// command_id
	binary.Write(buf, binary.BigEndian, msg.id)
	// command_status
	binary.Write(buf, binary.BigEndian, msg.stat)
	// sequence_number
	binary.Write(buf, binary.BigEndian, msg.seq)

	buf.Write(msg.body)
	e = buf.Flush()

	if e == nil && TraceMessage != nil {
		TraceMessage(Tx, msg.id, msg.stat, msg.seq, msg.body)
	}
	return
}

func readCString(buf *bytes.Buffer) (string, error) {
	b, e := buf.ReadBytes(0x00)
	if e != nil {
		b = []byte{0x00}
	}
	return string(b[:len(b)-1]), e
}

func writeCString(value []byte, buf *bytes.Buffer) {
	buf.Write(value)
	buf.WriteByte(0x00)
}

type OptionalParameters map[uint16]OctetData

func (p OptionalParameters) String() string {
	buf := new(strings.Builder)
	fmt.Fprint(buf, Indent, " optional_parameters:")
	for i, o := range p {
		if DecodeParameter == nil {
		} else if n, v, e := DecodeParameter(i, o); e == nil {
			fmt.Fprintf(buf, "\n%s %s %s: %v", Indent, Indent, n, v)
			continue
		}
		fmt.Fprintf(buf, "\n%s %s %#04x: 0x% x", Indent, Indent, i, o)
	}
	return buf.String()
}

func (p OptionalParameters) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	for i, o := range p {
		if DecodeParameter == nil {
		} else if n, v, e := DecodeParameter(i, o); e == nil {
			m[n] = v
			continue
		}
		m[fmt.Sprintf("%04x", i)] = o
	}
	return json.Marshal(m)
}

var DecodeParameter func(uint16, OctetData) (string, any, error) = nil

func (p *OptionalParameters) UnmarshalJSON(b []byte) (e error) {
	m := map[string]any{}
	if e = json.Unmarshal(b, &m); e != nil {
		return
	}
	r := map[uint16]OctetData{}
	for k, v := range m {
		if EncodeParameter == nil {
		} else if i, o, e := EncodeParameter(k, v); e == nil {
			r[i] = o
			continue
		}

		var b []byte
		if b, e = hex.DecodeString(k); e != nil || len(b) != 2 {
			e = errors.New("unknown parameter name " + k)
			return
		}
		i := uint16(b[0])
		i = (i << 8) | uint16(b[1])
		s, ok := v.(string)
		if !ok {
			e = errors.New("invalid parameter for " + k)
			return
		}
		if b, e = hex.DecodeString(s); e != nil {
			e = errors.New("invalid parameter for " + k)
			return
		}
		r[i] = OctetData(b)
	}
	*p = r
	return
}

var EncodeParameter func(string, any) (uint16, OctetData, error) = nil

func (p *OptionalParameters) readFrom(buf *bytes.Buffer) (e error) {
	r := map[uint16]OctetData{}
	for {
		var i, l uint16
		if e = binary.Read(buf, binary.BigEndian, &i); e == io.EOF {
			e = nil
			break
		} else if e != nil {
			return
		} else if e = binary.Read(buf, binary.BigEndian, &l); e != nil {
			return
		}
		v := make([]byte, int(l))
		if _, e = buf.Read(v); e != nil {
			return
		}
		r[i] = v
	}
	*p = r
	return
}

func (p OptionalParameters) writeTo(w *bytes.Buffer) {
	for i, o := range p {
		binary.Write(w, binary.BigEndian, i)
		binary.Write(w, binary.BigEndian, uint16(len(o)))
		w.Write(o)
	}
}

/*
type OptionalParameter struct {
	ID    uint16    `json:"id"`
	Value OctetData `json:"value"`
}

func (p OptionalParameter) String() string {
	if PrintParameter != nil {
		return PrintParameter(p)
	}
	return fmt.Sprintf("%#04x: 0x% x", p.ID, p.Value)
}

var PrintParameter func(OptionalParameter) string = nil

func (p *OptionalParameter) readFrom(buf *bytes.Buffer) (e error) {
	var l uint16
	if e = binary.Read(buf, binary.BigEndian, &(p.ID)); e != nil {
	} else if e = binary.Read(buf, binary.BigEndian, &l); e != nil {
	} else {
		p.Value = make([]byte, int(l))
		_, e = buf.Read(p.Value)
	}
	return
}

func (p OptionalParameter) writeTo(w *bytes.Buffer) {
	binary.Write(w, binary.BigEndian, p.ID)
	binary.Write(w, binary.BigEndian, uint16(len(p.Value)))
	w.Write(p.Value)
}

/*
func (p OptionalParameter) MarshalJSON() ([]byte, error) {
	if MarshalParameter != nil {
		return MarshalParameter(p)
	}
	return json.Marshal(struct {
		ID    uint16    `json:"id"`
		Value OctetData `json:"value"`
	}{
		ID:    p.ID,
		Value: p.Value})
}

var MarshalParameter func(OptionalParameter) ([]byte, error) = nil

func (p *OptionalParameter) UnmarshalJSON(b []byte) (e error) {
	if UnmarshalParameter != nil {
		*p, e = UnmarshalParameter(b)
		return
	}
	s := struct {
		ID    uint16    `json:"id"`
		Value OctetData `json:"value"`
	}{}
	if e = json.Unmarshal(b, &s); e == nil {
		p.ID = s.ID
		p.Value = s.Value
	}
	return
}

var UnmarshalParameter func([]byte) (OptionalParameter, error) = nil
*/

/*
func readTLV(buf *bytes.Buffer) (id uint16, value []byte, e error) {
	var l uint16
	if e = binary.Read(buf, binary.BigEndian, &id); e != nil {
	} else if e = binary.Read(buf, binary.BigEndian, &l); e != nil {
	} else {
		value = make([]byte, int(l))
		_, e = buf.Read(value)
	}
	return
}

func writeTLV(id uint16, value []byte, buf *bytes.Buffer) {
	if id != 0 {
		binary.Write(buf, binary.BigEndian, id)
		binary.Write(buf, binary.BigEndian, uint16(len(value)))
	}
	buf.Write(value)
}
*/

type OctetData []byte

func (d OctetData) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(d))
}

func (d *OctetData) UnmarshalJSON(b []byte) (e error) {
	s := ""
	var a []byte
	if e = json.Unmarshal(b, &s); e != nil {
	} else if a, e = hex.DecodeString(s); e == nil {
		*d = a
	}
	return
}

func writeAddr(nai teldata.NatureOfAddress, np teldata.NumberingPlan, addr string, buf *bytes.Buffer) {
	buf.WriteByte(byte(nai))
	buf.WriteByte(byte(np))
	writeCString([]byte(addr), buf)
}

func readAddr(buf *bytes.Buffer) (teldata.NatureOfAddress, teldata.NumberingPlan, string, error) {
	if nai, e := buf.ReadByte(); e != nil {
		return 0, 0, "", e
	} else if np, e := buf.ReadByte(); e != nil {
		return 0, 0, "", e
	} else if addr, e := readCString(buf); e != nil {
		return 0, 0, "", e
	} else {
		return teldata.NatureOfAddress(nai), teldata.NumberingPlan(np), addr, nil
	}
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
	}
	return errors.New("invalid Messaging Mode")
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
	}
	return errors.New("invalid Messaging Type")
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
	fmt.Fprintln(buf, Indent, Indent, "reply_path     :", c.ReplyPath)
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
	}
	return errors.New("invalid Delivery Receipt")
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
	fmt.Fprintln(buf, Indent, Indent, "intermadiate_delivery_notification:", r.IntermediateNotif)
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

func writeBool(b bool, buf *bytes.Buffer) {
	if b {
		buf.WriteByte(0x01)
	} else {
		buf.WriteByte(0x00)
	}
}

func readBool(buf *bytes.Buffer) (bool, error) {
	b, e := buf.ReadByte()
	return b == 0x01, e
}

type genericNack struct{}

func (*genericNack) CommandID() CommandID   { return GenericNack }
func (*genericNack) String() string         { return "" }
func (*genericNack) Marshal(byte) []byte    { return []byte{} }
func (*genericNack) Unmarshal([]byte) error { return nil }
