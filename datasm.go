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
	fmt.Fprintln(buf, "| service_type:       ", d.SvcType)
	fmt.Fprintln(buf, "| source_addr_ton:    ", d.SrcTON)
	fmt.Fprintln(buf, "| source_addr_npi:    ", d.SrcNPI)
	fmt.Fprintln(buf, "| source_addr:        ", d.SrcAddr)
	fmt.Fprintln(buf, "| dest_addr_ton:      ", d.DstTON)
	fmt.Fprintln(buf, "| dest_addr_npi:      ", d.DstNPI)
	fmt.Fprintln(buf, "| destination_addr:   ", d.DstAddr)
	fmt.Fprintln(buf, "| esm_class:          ", d.EsmClass)
	fmt.Fprintln(buf, "| registered_delivery:", d.RegisteredDelivery)
	fmt.Fprintln(buf, "| data_coding:        ", d.DataCoding)
	fmt.Fprint(buf, "| optional_parameters:")
	for t, v := range d.Param {
		fmt.Fprintf(buf, "\n| | %#04x: 0x% x", t, v)
	}
	return buf.String()
}

func (*DataSM) CommandID() CommandID { return DataSm }

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
	fmt.Fprintln(buf, "| id:", d.MessageID)
	fmt.Fprint(buf, "| optional_parameters:")
	for t, v := range d.Param {
		fmt.Fprintf(buf, "\n| | %#04x: %# x", t, v)
	}
	return buf.String()
}

func (*DataSM_resp) CommandID() CommandID { return DataSmResp }

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
