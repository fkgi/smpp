package smpp

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fkgi/teldata"
)

type DataSM struct {
	SvcType  string                  `json:"svc_type"`
	SrcTON   teldata.NatureOfAddress `json:"src_ton,omitempty"`
	SrcNPI   teldata.NumberingPlan   `json:"src_npi,omitempty"`
	SrcAddr  string                  `json:"src_addr,omitempty"`
	DstTON   teldata.NatureOfAddress `json:"dst_ton"`
	DstNPI   teldata.NumberingPlan   `json:"dst_npi"`
	DstAddr  string                  `json:"dst_addr"`
	EsmClass esmClass                `json:"esm_class"`

	RegisteredDelivery registeredDelivery `json:"registered_delivery"`
	DataCoding         byte               `json:"data_coding"`

	Param OptionalParameters `json:"options,omitempty"`
}

func (d *DataSM) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, "service_type       :", d.SvcType)
	fmt.Fprintln(buf, Indent, "source_addr_ton    :", d.SrcTON)
	fmt.Fprintln(buf, Indent, "source_addr_npi    :", d.SrcNPI)
	fmt.Fprintln(buf, Indent, "source_addr        :", d.SrcAddr)
	fmt.Fprintln(buf, Indent, "dest_addr_ton      :", d.DstTON)
	fmt.Fprintln(buf, Indent, "dest_addr_npi      :", d.DstNPI)
	fmt.Fprintln(buf, Indent, "destination_addr   :", d.DstAddr)
	fmt.Fprintln(buf, Indent, "esm_class          :", d.EsmClass)
	fmt.Fprintln(buf, Indent, "registered_delivery:", d.RegisteredDelivery)
	fmt.Fprintln(buf, Indent, "data_coding        :", d.DataCoding)
	fmt.Fprint(buf, d.Param)
	return buf.String()
}

func (*DataSM) CommandID() CommandID { return DataSm }

func (d *DataSM) Marshal(v byte) []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.SvcType), w)
	writeAddr(d.SrcTON, d.SrcNPI, d.SrcAddr, w)
	writeAddr(d.DstTON, d.DstNPI, d.DstAddr, w)
	d.EsmClass.writeTo(w)
	d.RegisteredDelivery.writeTo(w)
	w.WriteByte(d.DataCoding)
	if v >= 0x34 {
		d.Param.writeTo(w)
	}
	return w.Bytes()
}

func (d *DataSM) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.SvcType, e = readCString(buf); e != nil {
	} else if d.SrcTON, d.SrcNPI, d.SrcAddr, e = readAddr(buf); e != nil {
	} else if d.DstTON, d.DstNPI, d.DstAddr, e = readAddr(buf); e != nil {
	} else if e = d.EsmClass.readFrom(buf); e != nil {
	} else if e = d.RegisteredDelivery.readFrom(buf); e != nil {
	} else if d.DataCoding, e = buf.ReadByte(); e != nil {
	} else {
		d.Param = OptionalParameters{}
		e = d.Param.readFrom(buf)
	}
	return
}

type DataSM_resp struct {
	MessageID string             `json:"id"`
	Param     OptionalParameters `json:"options,omitempty"`
}

func (d *DataSM_resp) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, "id:", d.MessageID)
	fmt.Fprint(buf, d.Param)
	return buf.String()
}

func (*DataSM_resp) CommandID() CommandID { return DataSmResp }

func (d *DataSM_resp) Marshal(v byte) []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.MessageID), w)
	if v >= 0x34 {
		d.Param.writeTo(w)
	}
	return w.Bytes()
}

func (d *DataSM_resp) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.MessageID, e = readCString(buf); e == nil {
		d.Param = OptionalParameters{}
		e = d.Param.readFrom(buf)
	}
	return
}
