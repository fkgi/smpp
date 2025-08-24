package smpp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

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

type genericNack struct{}

func (*genericNack) CommandID() CommandID   { return GenericNack }
func (*genericNack) String() string         { return "" }
func (*genericNack) Marshal(byte) []byte    { return []byte{} }
func (*genericNack) Unmarshal([]byte) error { return nil }
