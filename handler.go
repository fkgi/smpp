package smpp

import (
	"bytes"
	"time"
)

var DeliverHandler = func(info BindInfo, body []byte) (uint32, []byte) {
	return 0, []byte{}
}

var SubmitHandler = func(info BindInfo, body []byte) (uint32, []byte) {
	w := new(bytes.Buffer)
	w.WriteString(time.Now().String())
	w.WriteByte(0)
	return 0, w.Bytes()
}

var RequestHandler = func(info BindInfo, pdu PDU) (uint32, PDU) {
	switch pdu.(type) {
	case *DataSM:
		return 0, &DataSM_resp{
			MessageID: "random id",
		}
	case *DeliverSM:
		return 0, &DeliverSM_resp{
			MessageID: "random id",
		}
	}
	return 0x00000003, nil
}

var DataRespHandler = func(info BindInfo, stat uint32, body []byte) {
}

/*

func DialTransmitter(addr string, info BindInfo) (t Transmitter, e error) {
	if len(info.SystemID) > 15 {
		info.SystemID = info.SystemID[:16]
	}
	if len(info.Password) > 8 {
		info.Password = info.Password[:9]
	}
	if len(info.SystemType) > 12 {
		info.SystemType = info.SystemType[:13]
	}
	t.con, e = net.Dial("tcp", addr)
	if e != nil {
		return
	}
	t.seq = newSequence()

	w := new(bytes.Buffer)
	w.WriteString(info.SystemID)
	w.WriteByte(0)
	w.WriteString(info.Password)
	w.WriteByte(0)
	w.WriteString(info.SystemType)
	w.WriteByte(0)
	// interface_version
	w.WriteByte(0x34)
	w.WriteByte(info.TypeOfNumber)
	w.WriteByte(info.NumberingPlan)
	w.WriteString(info.AddressRange)
	w.WriteByte(0)

	e = writePDU(t.con, 0x00000002, 0, t.seq.next(), w.Bytes())
	if e != nil {
		return
	}

	mid, _, num, _, e := readPDU(t.con)
	if e != nil {
	} else if mid != 0x80000002 {
		e = errors.New("invalid response")
	} else if num != 1 {
		e = errors.New("invalid response")
	}

	return
}

func (t Transmitter) Submit() error {
	u := utf16.Encode([]rune("てすと"))
	ud := make([]byte, len(u)*2)
	for i, c := range u {
		ud[i*2] = byte((c >> 8) & 0xff)
		ud[i*2+1] = byte(c & 0xff)
	}

	w := new(bytes.Buffer)
	// service_type
	w.WriteString("TEST")
	w.WriteByte(0)
	// source_addr_ton
	w.WriteByte(0)
	// source_addr_npi
	w.WriteByte(0)
	// source_addr
	w.WriteByte(0)
	// dest_addr_ton
	w.WriteByte(0x01)
	// dest_addr_npi
	w.WriteByte(0x01)
	// destination_addr
	w.WriteString("819011112222")
	w.WriteByte(0)
	// esm_class
	w.WriteByte(0x00)
	// protocol_id
	w.WriteByte(0x00)
	// priority_flag
	w.WriteByte(0x00)
	// schedule_delivery_time
	w.WriteByte(0x00)
	// validity_period
	w.WriteByte(0x00)
	// registered_delivery
	w.WriteByte(0x00)
	// replace_if_present_flag
	w.WriteByte(0x00)
	// data_coding
	w.WriteByte(0x08)
	// sm_default_msg_id
	w.WriteByte(0x00)
	// sm_length
	w.WriteByte(byte(len(ud)))
	// short_message
	w.Write(ud)

	e := writePDU(t.con, 0x00000004, 0, t.seq.next(), w.Bytes())
	if e != nil {
		return e
	}

	mid, _, _, _, e := readPDU(t.con)
	if e != nil {
	} else if mid != 0x80000004 {
		return errors.New("invalid response")
	}
	return nil
}

func (t Transmitter) Close() error {
	e := writePDU(t.con, 0x00000006, 0, t.seq.next(), nil)
	if e != nil {
		return e
	}
	mid, _, _, _, e := readPDU(t.con)
	if e != nil {
	} else if mid != 0x80000006 {
		return errors.New("invalid response")
	}
	return nil
}

func (t Transmitter) Enquire() error {
	e := writePDU(t.con, 0x00000015, 0, t.seq.next(), nil)
	if e != nil {
		return e
	}
	mid, _, _, _, e := readPDU(t.con)
	if e != nil {
	} else if mid != 0x80000015 {
		return errors.New("invalid response")
	}
	return nil
}

*/
