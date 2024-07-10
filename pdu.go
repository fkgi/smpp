package smpp

import (
	"bytes"
	"io"
)

type PDU interface {
	CommandID() uint32
	Marshal() []byte
	Unmarshal([]byte) error
}

type Request interface {
	PDU
}

type Response interface {
	PDU
	CommandStatus() uint32
}

type DataSM struct {
	SvcType  string
	SrcTON   byte
	SrcNPI   byte
	SrcAddr  string
	DstTON   byte
	DstNPI   byte
	DstAddr  string
	EsmClass byte

	RegisteredDelivery byte
	DataCoding         byte
	Param              map[uint16][]byte
}

func (*DataSM) CommandID() uint32 {
	return 0x00000103
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
	Status    uint32
	MessageID string
	Param     map[uint16][]byte
}

func (*DataSM_resp) CommandID() uint32 {
	return 0x80000103
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

type DeliverSM struct {
	SvcType  string
	SrcTON   byte
	SrcNPI   byte
	SrcAddr  string
	DstTON   byte
	DstNPI   byte
	DstAddr  string
	EsmClass byte

	ProtocolId           byte
	PriorityFlag         byte
	ScheduleDeliveryTime string
	ValidityPeriod       string
	RegisteredDelivery   byte
	ReplaceIfPresentFlag byte
	DataCoding           byte
	SmDefaultMsgId       byte
	// SmLength            byte
	ShortMessage []byte

	Param map[uint16][]byte
}

func (*DeliverSM) CommandID() uint32 {
	return 0x00000005
}

func (d *DeliverSM) Marshal() []byte {
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

func (d *DeliverSM) Unmarshal(data []byte) (e error) {
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

type DeliverSM_resp struct {
	MessageID string
}

func (*DeliverSM_resp) CommandID() uint32 {
	return 0x80000005
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
