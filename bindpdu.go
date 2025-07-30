package smpp

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type bindReq struct {
	cmd        CommandID
	SystemID   string `json:"system_id"`
	Password   string `json:"passsword"`
	SystemType string `json:"system_type"`
	Version    byte   `json:"interface_version"`
	AddrTON    byte   `json:"addr_ton"`
	AddrNPI    byte   `json:"addr_npi"`
	AddrRange  string `json:"address_range"`
}

func (d *bindReq) CommandID() CommandID { return d.cmd }

func (d *bindReq) String() string {
	buf := new(strings.Builder)
	fmt.Fprintln(buf, "| system_id:         ", d.SystemID)
	fmt.Fprintln(buf, "| passsword:         ", d.Password)
	fmt.Fprintln(buf, "| system_type:       ", d.SystemType)
	fmt.Fprintln(buf, "| interface_version: ", d.Version)
	fmt.Fprintln(buf, "| addr_ton:          ", d.AddrTON)
	fmt.Fprintln(buf, "| addr_npi:          ", d.AddrNPI)
	fmt.Fprintln(buf, "| address_range:     ", d.AddrRange)
	return buf.String()
}

func (d *bindReq) Marshal() []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.SystemID), w)
	writeCString([]byte(d.Password), w)
	writeCString([]byte(d.SystemType), w)
	w.WriteByte(d.Version)
	w.WriteByte(d.AddrTON)
	w.WriteByte(d.AddrNPI)
	writeCString([]byte(d.AddrRange), w)
	return w.Bytes()
}

func (d *bindReq) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.SystemID, e = readCString(buf); e != nil {
	} else if d.Password, e = readCString(buf); e != nil {
	} else if d.SystemType, e = readCString(buf); e != nil {
	} else if d.Version, e = buf.ReadByte(); e != nil {
	} else if d.AddrTON, e = buf.ReadByte(); e != nil {
	} else if d.AddrNPI, e = buf.ReadByte(); e != nil {
	} else {
		d.AddrRange, e = readCString(buf)
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
	fmt.Fprintln(buf, "| system_id:            ", d.SystemID)
	fmt.Fprintln(buf, "| sc_interface_version: ", d.Version)
	return buf.String()
}

func (d *bindRes) Marshal() []byte {
	w := new(bytes.Buffer)
	writeCString([]byte(d.SystemID), w)
	writeTLV(0x0210, []byte{d.Version}, w)
	return w.Bytes()
}

func (d *bindRes) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	if d.SystemID, e = readCString(buf); e != nil {
		return
	}
	for {
		t, v, e2 := readTLV(buf)
		if e2 == io.EOF {
			break
		}
		if e2 != nil {
			e = e2
			break
		}

		switch t {
		case 0x0210:
			d.Version = v[0]
		}
	}
	return
}

type unbindReq struct{}

func (*unbindReq) CommandID() CommandID   { return Unbind }
func (*unbindReq) String() string         { return "" }
func (*unbindReq) Marshal() []byte        { return []byte{} }
func (*unbindReq) Unmarshal([]byte) error { return nil }

type unbindRes struct{}

func (*unbindRes) CommandID() CommandID   { return UnbindResp }
func (*unbindRes) String() string         { return "" }
func (*unbindRes) Marshal() []byte        { return []byte{} }
func (*unbindRes) Unmarshal([]byte) error { return nil }

type enquireReq struct{}

func (*enquireReq) CommandID() CommandID   { return EnquireLink }
func (*enquireReq) String() string         { return "" }
func (*enquireReq) Marshal() []byte        { return []byte{} }
func (*enquireReq) Unmarshal([]byte) error { return nil }

type enquireRes struct{}

func (*enquireRes) CommandID() CommandID   { return EnquireLinkResp }
func (*enquireRes) String() string         { return "" }
func (*enquireRes) Marshal() []byte        { return []byte{} }
func (*enquireRes) Unmarshal([]byte) error { return nil }
