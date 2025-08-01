package smpp

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

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
	fmt.Fprintln(buf, Indent, "sm_length              :", len(d.ShortMessage))
	if len(d.ShortMessage) != 0 {
		fmt.Fprintf(buf, "%s short_message          :0x% x\n", Indent, d.ShortMessage)
	} else {
		fmt.Fprintln(buf, Indent, "short_message          :")
	}
	fmt.Fprint(buf, Indent, " optional_parameters:")
	for t, v := range d.Param {
		fmt.Fprintf(buf, "\n%s %s %#04x: 0x% x", Indent, Indent, t, v)
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

type SubmitSM struct {
	smPDU
}

func (*SubmitSM) CommandID() CommandID { return SubmitSm }

type SubmitSM_resp struct {
	MessageID string `json:"id,omitempty"`
}

func (d *SubmitSM_resp) String() string     { return Indent + " id: " + d.MessageID }
func (*SubmitSM_resp) CommandID() CommandID { return SubmitSmResp }

func (d *SubmitSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	if len(d.MessageID) != 0 {
		writeCString([]byte(d.MessageID), &w)
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

func (d *DeliverSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	writeCString([]byte{}, &w)
	return w.Bytes()
}

func (d *DeliverSM_resp) Unmarshal(data []byte) (e error) {
	return
}
