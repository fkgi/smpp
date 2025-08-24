package smpp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"unicode/utf16"

	"github.com/fkgi/sms"
)

type UserData struct {
	Text string        `json:"text,omitempty"`
	UDH  []UserDataHdr `json:"hdr,omitempty"`
}

type UserDataHdr struct {
	Key byte      `json:"key"`
	Val OctetData `json:"value"`
}

func (u UserData) String() string {
	w := new(bytes.Buffer)
	for _, h := range u.UDH {
		fmt.Fprintf(w, "\n%s %s UDH(%x)=%x", Indent, Indent, h.Key, h.Val)
	}
	if len(u.Text) != 0 {
		fmt.Fprintf(w, "\n%s %s %s", Indent, Indent, u.Text)
	}
	return w.String()
}

// Set8bitData set binary data as UD
func (u *UserData) Set8bitData(d []byte) {
	if u != nil && len(d) != 0 {
		u.Text = base64.StdEncoding.EncodeToString(d)
	}
}

// Get8bitData set binary data as UD
func (u UserData) Get8bitData() ([]byte, error) {
	return base64.StdEncoding.DecodeString(u.Text)
}

func (u *UserData) unmarshal(ud []byte, dc byte, h bool) error {
	o := 0
	l := len(ud)
	if h {
		o = int(ud[0]+1) * 8
		l -= o / 7
		o %= 7
		if o != 0 {
			o = 7 - o
			l--
		}

		u.UDH = []UserDataHdr{}
		for buf := bytes.NewBuffer(ud[1 : ud[0]+1]); buf.Len() != 0; {
			k, _ := buf.ReadByte()
			l, _ := buf.ReadByte()
			v := make([]byte, l)
			buf.Read(v)
			udh := UserDataHdr{
				Key: k,
				Val: OctetData(v)}
			u.UDH = append(u.UDH, udh)
		}
		ud = ud[ud[0]+1:]
	}

	switch dc {
	case 0x00:
		s := sms.UnmarshalGSM7bitString(o, l, ud)
		u.Text = s.String()
	case 0x08:
		s := make([]uint16, len(ud)/2)
		for i := range s {
			s[i] = uint16(ud[2*i])<<8 | uint16(ud[2*i+1])
		}
		u.Text = string(utf16.Decode(s))
	default:
		u.Text = base64.StdEncoding.EncodeToString(ud)
	}
	return nil
}

func (u UserData) marshal(dc byte) []byte {
	w := bytes.Buffer{}
	for _, u := range u.UDH {
		w.WriteByte(u.Key)
		w.WriteByte(byte(len(u.Val)))
		w.Write(u.Val)
	}

	d := w.Bytes()
	w = bytes.Buffer{}
	if len(d) != 0 {
		w.WriteByte(byte(len(d)))
		w.Write(d)
	}

	switch dc {
	case 0x00:
		o := (len(d) * 8) % 7
		if o != 0 {
			o = 7 - o
		}
		s, _ := sms.StringToGSM7bit(u.Text)
		w.Write(s.Marshal(o))
	case 0x08:
		u := utf16.Encode([]rune(u.Text))
		ud := make([]byte, len(u)*2)
		for i, c := range u {
			ud[i*2] = byte((c >> 8) & 0xff)
			ud[i*2+1] = byte(c & 0xff)
		}
		w.Write(ud)
	default:
		ud, e := base64.StdEncoding.DecodeString(u.Text)
		if e != nil {
			ud = []byte{}
		}
		w.Write(ud)
	}

	return w.Bytes()
}
