package smpp

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

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

	Param map[uint16]OctetData `json:"options,omitempty"`
}

func (d *DataSM) String() string {
	buf := new(strings.Builder)
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
	fmt.Fprint(buf, Indent, " optional_parameters:")
	for t, v := range d.Param {
		fmt.Fprintf(buf, "\n%s %s %#04x: 0x% x", Indent, Indent, t, v)
	}
	return buf.String()
}

func (*DataSM) CommandID() CommandID { return DataSm }

func (d *DataSM) Marshal(v byte) []byte {
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
	if v >= 0x34 {
		for k, v := range d.Param {
			writeTLV(k, v, &w)
		}
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

type DataSM_resp struct {
	MessageID string            `json:"id"`
	Param     map[uint16][]byte `json:"options,omitempty"`
}

func (d *DataSM_resp) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf, Indent, "id:", d.MessageID)
	fmt.Fprint(buf, Indent, " optional_parameters:")
	for t, v := range d.Param {
		fmt.Fprintf(buf, "\n%s %s %#04x: %# x", Indent, Indent, t, v)
	}
	return buf.String()
}

func (*DataSM_resp) CommandID() CommandID { return DataSmResp }

func (d *DataSM_resp) Marshal(v byte) []byte {
	w := bytes.Buffer{}
	writeCString([]byte(d.MessageID), &w)
	if v >= 0x34 {
		for k, v := range d.Param {
			writeTLV(k, v, &w)
		}
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
