package smpp

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fkgi/teldata"
)

type bindReq struct {
	cmd        CommandID
	SystemID   string                  `json:"system_id"`
	Password   string                  `json:"passsword"`
	SystemType string                  `json:"system_type"`
	Version    byte                    `json:"interface_version"`
	AddrTON    teldata.NatureOfAddress `json:"addr_ton"`
	AddrNPI    teldata.NumberingPlan   `json:"addr_npi"`
	AddrRange  string                  `json:"address_range"`
}

func (d *bindReq) CommandID() CommandID { return d.cmd }

func (d *bindReq) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, "system_id         :", d.SystemID)
	fmt.Fprintln(buf, Indent, "passsword         :", d.Password)
	fmt.Fprintln(buf, Indent, "system_type       :", d.SystemType)
	fmt.Fprintln(buf, Indent, "interface_version :", d.Version)
	fmt.Fprintln(buf, Indent, "addr_ton          :", d.AddrTON)
	fmt.Fprintln(buf, Indent, "addr_npi          :", d.AddrNPI)
	fmt.Fprintln(buf, Indent, "address_range     :", d.AddrRange)
	return buf.String()
}

func (d *bindReq) Marshal(byte) []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.SystemID), w)
	writeCString([]byte(d.Password), w)
	writeCString([]byte(d.SystemType), w)
	w.WriteByte(d.Version)
	writeAddr(d.AddrTON, d.AddrNPI, d.AddrRange, w)
	return w.Bytes()
}

func (d *bindReq) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.SystemID, e = readCString(buf); e != nil {
	} else if d.Password, e = readCString(buf); e != nil {
	} else if d.SystemType, e = readCString(buf); e != nil {
	} else if d.Version, e = buf.ReadByte(); e != nil {
	} else {
		d.AddrTON, d.AddrNPI, d.AddrRange, e = readAddr(buf)
	}
	return
}

type bindRes struct {
	cmd      CommandID
	SystemID string `json:"system_id"`
	Version  byte   `json:"sc_interface_version,omitempty"`
}

func (d *bindRes) CommandID() CommandID { return d.cmd }

func (d *bindRes) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, Indent, "system_id           :", d.SystemID)
	fmt.Fprintln(buf, Indent, "sc_interface_version:", d.Version)
	return buf.String()
}

func (d *bindRes) Marshal(v byte) []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.SystemID), w)
	if v >= 0x34 {
		w.Write([]byte{0x02, 0x10, 0x00, 0x01, d.Version})
	}
	return w.Bytes()
}

func (d *bindRes) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.SystemID, e = readCString(buf); e != nil {
		return
	}
	p := OptionalParameters{}
	if e = p.readFrom(buf); e != nil {
		return
	}
	if v, ok := p[0x0210]; ok && len(v) == 1 {
		d.Version = v[0]
	}
	return
}

type unbindReq struct{}

func (*unbindReq) CommandID() CommandID   { return Unbind }
func (*unbindReq) String() string         { return "" }
func (*unbindReq) Marshal(byte) []byte    { return []byte{} }
func (*unbindReq) Unmarshal([]byte) error { return nil }

type unbindRes struct{}

func (*unbindRes) CommandID() CommandID   { return UnbindResp }
func (*unbindRes) String() string         { return "" }
func (*unbindRes) Marshal(byte) []byte    { return []byte{} }
func (*unbindRes) Unmarshal([]byte) error { return nil }

type enquireReq struct{}

func (*enquireReq) CommandID() CommandID   { return EnquireLink }
func (*enquireReq) String() string         { return "" }
func (*enquireReq) Marshal(byte) []byte    { return []byte{} }
func (*enquireReq) Unmarshal([]byte) error { return nil }

type enquireRes struct{}

func (*enquireRes) CommandID() CommandID   { return EnquireLinkResp }
func (*enquireRes) String() string         { return "" }
func (*enquireRes) Marshal(byte) []byte    { return []byte{} }
func (*enquireRes) Unmarshal([]byte) error { return nil }
