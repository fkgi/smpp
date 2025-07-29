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
)

type pdu interface {
	CommandID() CommandID
	Marshal() []byte
	Unmarshal([]byte) error
	fmt.Stringer
}

type Request interface {
	pdu
	MakeResponse() Response
}

type Response interface {
	pdu
	// CommandStatus() StatusCode
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
	return string(b[:len(b)-1]), e
}

func writeCString(value []byte, buf *bytes.Buffer) {
	buf.Write(value)
	buf.WriteByte(0x00)
}

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

type OctetData []byte

func (d OctetData) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(d))
}

func (d *OctetData) UnmarshalJSON(b []byte) (e error) {
	s := ""
	if e = json.Unmarshal(b, &s); e != nil {
	} else if a, e := hex.DecodeString(s); e == nil {
		*d = a
	}
	return
}

type smPDU struct {
	SvcType  string `json:"svc_type,omitempty"`
	SrcTON   byte   `json:"src_ton,omitempty"`
	SrcNPI   byte   `json:"src_npi,omitempty"`
	SrcAddr  string `json:"src_addr,omitempty"`
	DstTON   byte   `json:"dst_ton"`
	DstNPI   byte   `json:"dst_npi"`
	DstAddr  string `json:"dst_addr"`
	EsmClass byte   `json:"esm_class"`

	ProtocolId           byte   `json:"protocol_id"`
	PriorityFlag         byte   `json:"priority_flag"`
	ScheduleDeliveryTime string `json:"schedule_delivery_time,omitempty"`
	ValidityPeriod       string `json:"validity_period,omitempty"`
	RegisteredDelivery   byte   `json:"registered_delivery"`
	ReplaceIfPresentFlag byte   `json:"replace_if_present_flag,omitempty"`
	DataCoding           byte   `json:"data_coding"`
	SmDefaultMsgId       byte   `json:"sm_default_sm_id,omitempty"`
	// SmLength            byte
	ShortMessage OctetData `json:"short_message,omitempty"`

	Param map[uint16]OctetData `json:"options,omitempty"`
}

func (d *smPDU) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf, "| service_type:           ", d.SvcType)
	fmt.Fprintln(buf, "| source_addr_ton:        ", d.SrcTON)
	fmt.Fprintln(buf, "| source_addr_npi:        ", d.SrcNPI)
	fmt.Fprintln(buf, "| source_addr:            ", d.SrcAddr)
	fmt.Fprintln(buf, "| dest_addr_ton:          ", d.DstTON)
	fmt.Fprintln(buf, "| dest_addr_npi:          ", d.DstNPI)
	fmt.Fprintln(buf, "| destination_addr:       ", d.DstAddr)
	fmt.Fprintln(buf, "| esm_class:              ", d.EsmClass)
	fmt.Fprintln(buf, "| protocol_id:            ", d.ProtocolId)
	fmt.Fprintln(buf, "| priority_flag:          ", d.PriorityFlag)
	fmt.Fprintln(buf, "| schedule_delivery_time: ", d.ScheduleDeliveryTime)
	fmt.Fprintln(buf, "| validity_period:        ", d.ValidityPeriod)
	fmt.Fprintln(buf, "| registered_delivery:    ", d.RegisteredDelivery)
	fmt.Fprintln(buf, "| replace_if_present_flag:", d.ReplaceIfPresentFlag)
	fmt.Fprintln(buf, "| data_coding:            ", d.DataCoding)
	fmt.Fprintln(buf, "| sm_default_msg_id:      ", d.SmDefaultMsgId)
	fmt.Fprintln(buf, "| sm_length:              ", len(d.ShortMessage))
	fmt.Fprintf(buf, "| short_message:          0x% x\n", d.ShortMessage)
	fmt.Fprint(buf, "| optional_parameters:")
	for t, v := range d.Param {
		fmt.Fprintf(buf, "\n| | %#04x: 0x% x", t, v)
	}
	return buf.String()
}

func (d *smPDU) Marshal() []byte {
	w := bytes.Buffer{}

	writeCString([]byte(d.SvcType), &w)
	w.WriteByte(d.SrcTON)
	w.WriteByte(d.SrcNPI)
	writeCString([]byte(d.SrcAddr), &w)
	w.WriteByte(d.DstTON)
	w.WriteByte(d.DstNPI)
	writeCString([]byte(d.DstAddr), &w)
	w.WriteByte(d.EsmClass)
	w.WriteByte(d.ProtocolId)
	w.WriteByte(d.PriorityFlag)
	writeCString([]byte(d.ScheduleDeliveryTime), &w)
	writeCString([]byte(d.ValidityPeriod), &w)
	w.WriteByte(d.RegisteredDelivery)
	w.WriteByte(d.ReplaceIfPresentFlag)
	w.WriteByte(d.DataCoding)
	w.WriteByte(d.SmDefaultMsgId)
	// w.WriteByte(d.SmLength)
	w.WriteByte(byte(len(d.ShortMessage)))
	w.Write(d.ShortMessage)

	for k, v := range d.Param {
		writeTLV(k, v, &w)
	}

	return w.Bytes()
}

func (d *smPDU) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	var l byte
	if d.SvcType, e = readCString(buf); e != nil {
	} else if d.SrcTON, e = buf.ReadByte(); e != nil {
	} else if d.SrcNPI, e = buf.ReadByte(); e != nil {
	} else if d.SrcAddr, e = readCString(buf); e != nil {
	} else if d.DstTON, e = buf.ReadByte(); e != nil {
	} else if d.DstNPI, e = buf.ReadByte(); e != nil {
	} else if d.DstAddr, e = readCString(buf); e != nil {
	} else if d.EsmClass, e = buf.ReadByte(); e != nil {
	} else if d.ProtocolId, e = buf.ReadByte(); e != nil {
	} else if d.PriorityFlag, e = buf.ReadByte(); e != nil {
	} else if d.ScheduleDeliveryTime, e = readCString(buf); e != nil {
	} else if d.ValidityPeriod, e = readCString(buf); e != nil {
	} else if d.RegisteredDelivery, e = buf.ReadByte(); e != nil {
	} else if d.ReplaceIfPresentFlag, e = buf.ReadByte(); e != nil {
	} else if d.DataCoding, e = buf.ReadByte(); e != nil {
	} else if d.SmDefaultMsgId, e = buf.ReadByte(); e != nil {
	} else if l, e = buf.ReadByte(); e != nil {
	} else {
		d.ShortMessage = make([]byte, int(l))
		_, e = buf.Read(d.ShortMessage)
	}

	if e == nil {
		d.Param = make(map[uint16]OctetData)
		for {
			t, v, e2 := readTLV(buf)
			if e2 == io.EOF {
				break
			}
			if e2 != nil {
				e = e2
				break
			}
			d.Param[t] = v
		}
	}
	return
}
