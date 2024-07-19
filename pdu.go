package smpp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type PDU interface {
	CommandID() CommandID
	Marshal() []byte
	Unmarshal([]byte) error
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

	if RxMessageNotify != nil {
		RxMessageNotify(msg.id, msg.stat, msg.seq, msg.body)
	}
	return
}

func (b *Bind) writePDU(id CommandID, stat, seq uint32, body []byte) (e error) {
	if body == nil {
		body = []byte{}
	}
	buf := bufio.NewWriter(b.con)

	// command_length
	binary.Write(buf, binary.BigEndian, uint32(len(body)+16))
	// command_id
	binary.Write(buf, binary.BigEndian, id)
	// command_status
	binary.Write(buf, binary.BigEndian, stat)
	// sequence_number
	binary.Write(buf, binary.BigEndian, seq)

	buf.Write(body)
	e = buf.Flush()

	if e == nil && TxMessageNotify != nil {
		TxMessageNotify(id, stat, seq, body)
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

type Request interface {
	PDU
}

type Response interface {
	PDU
	CommandStatus() uint32
}

type DataSM struct {
	SvcType  string `json:"svc_type"`
	SrcTON   byte   `json:"src_ton,omitempty"`
	SrcNPI   byte   `json:"src_npi,omitempty"`
	SrcAddr  string `json:"src_addr,omitempty"`
	DstTON   byte   `json:"dst_ton"`
	DstNPI   byte   `json:"dst_npi"`
	DstAddr  string `json:"dst_addr"`
	EsmClass byte   `json:"esm_class"`

	RegisteredDelivery byte `json:"registered_delivery"`
	DataCoding         byte `json:"data_coding"`

	Param map[uint16][]byte `json:"options,omitempty"`
}

func (*DataSM) CommandID() CommandID {
	return DataSm
}

func (d *DataSM) Marshal() []byte {
	w := bytes.Buffer{}

	writeCString([]byte(d.SvcType), &w)
	w.WriteByte(d.SrcTON)
	w.WriteByte(d.SrcNPI)
	writeCString([]byte(d.SrcAddr), &w)
	w.WriteByte(d.DstTON)
	w.WriteByte(d.DstNPI)
	writeCString([]byte(d.DstAddr), &w)
	w.WriteByte(d.EsmClass)
	w.WriteByte(d.RegisteredDelivery)
	w.WriteByte(d.DataCoding)

	for k, v := range d.Param {
		writeTLV(k, v, &w)
	}

	return w.Bytes()
}

func (d *DataSM) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.SvcType, e = readCString(buf); e != nil {
	} else if d.SrcTON, e = buf.ReadByte(); e != nil {
	} else if d.SrcNPI, e = buf.ReadByte(); e != nil {
	} else if d.SrcAddr, e = readCString(buf); e != nil {
	} else if d.DstTON, e = buf.ReadByte(); e != nil {
	} else if d.DstNPI, e = buf.ReadByte(); e != nil {
	} else if d.DstAddr, e = readCString(buf); e != nil {
	} else if d.EsmClass, e = buf.ReadByte(); e != nil {
	} else if d.RegisteredDelivery, e = buf.ReadByte(); e != nil {
	} else if d.DataCoding, e = buf.ReadByte(); e != nil {
	} else {
		d.Param = make(map[uint16][]byte)
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

type DataSM_resp struct {
	Status    uint32 `json:"status"`
	MessageID string `json:"id"`

	Param map[uint16][]byte `json:"options,omitempty"`
}

func (*DataSM_resp) CommandID() CommandID {
	return DataSmResp
}

func (d *DataSM_resp) Marshal() []byte {
	w := bytes.Buffer{}

	writeCString([]byte(d.MessageID), &w)

	for k, v := range d.Param {
		writeTLV(k, v, &w)
	}

	return w.Bytes()
}

func (d *DataSM_resp) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.MessageID, e = readCString(buf); e != nil {
	} else {
		d.Param = make(map[uint16][]byte)
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

func (d *DataSM_resp) CommandStatus() uint32 {
	return d.Status
}

type smPDU struct {
	SvcType  string `json:"svc_type"`
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
	ShortMessage []byte `json:"short_message"`

	Param map[uint16][]byte `json:"options,omitempty"`
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
		d.Param = make(map[uint16][]byte)
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

type SubmitSM struct {
	smPDU
}

func (*SubmitSM) CommandID() CommandID {
	return SubmitSm
}

func (d *SubmitSM) Marshal() []byte {
	return d.smPDU.Marshal()
}

func (d *SubmitSM) Unmarshal(data []byte) (e error) {
	return d.smPDU.Unmarshal(data)
}

type SubmitSM_resp struct {
	Status    uint32 `json:"status"`
	MessageID string `json:"id"`
}

func (*SubmitSM_resp) CommandID() CommandID {
	return SubmitSmResp
}

func (d *SubmitSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	writeCString([]byte(d.MessageID), &w)
	return w.Bytes()
}

func (d *SubmitSM_resp) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	d.MessageID, e = readCString(buf)
	return
}

func (d *SubmitSM_resp) CommandStatus() uint32 {
	return d.Status
}

type DeliverSM struct {
	smPDU
}

func (*DeliverSM) CommandID() CommandID {
	return DeliverSm
}

func (d *DeliverSM) Marshal() []byte {
	return d.smPDU.Marshal()
}

func (d *DeliverSM) Unmarshal(data []byte) (e error) {
	return d.smPDU.Unmarshal(data)
}

type DeliverSM_resp struct {
	Status    uint32 `json:"status"`
	MessageID string `json:"id,omitempty"`
}

func (*DeliverSM_resp) CommandID() CommandID {
	return DeliverSmResp
}

func (d *DeliverSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	writeCString([]byte(d.MessageID), &w)
	return w.Bytes()
}

func (d *DeliverSM_resp) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	d.MessageID, e = readCString(buf)
	return
}

func (d *DeliverSM_resp) CommandStatus() uint32 {
	return d.Status
}
