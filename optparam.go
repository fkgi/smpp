package smpp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type OptionalParameters map[uint16][]byte

func (p OptionalParameters) String() string {
	buf := new(strings.Builder)
	fmt.Fprint(buf, Indent, " optional_parameters:")
	for i, o := range p {
		if DecodeParameter == nil {
		} else if n, v, e := DecodeParameter(i, o); e == nil {
			fmt.Fprintf(buf, "\n%s %s %s: %v", Indent, Indent, n, v)
			continue
		}
		fmt.Fprintf(buf, "\n%s %s unknown(%s): 0x% x",
			Indent, Indent, IdToHexString(i), o)
	}
	return buf.String()
}

func (p OptionalParameters) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	for i, o := range p {
		if DecodeParameter == nil {
		} else if n, v, e := DecodeParameter(i, o); e == nil {
			m[n] = v
			continue
		}
		m[IdToHexString(i)] = hex.EncodeToString(o)
	}
	return json.Marshal(m)
}

var DecodeParameter func(uint16, []byte) (string, any, error) = nil

func (p *OptionalParameters) UnmarshalJSON(b []byte) (e error) {
	m := map[string]any{}
	if e = json.Unmarshal(b, &m); e != nil {
		return
	}

	r := map[uint16][]byte{}
	for k, v := range m {
		if EncodeParameter == nil {
		} else if i, o, e := EncodeParameter(k, v); e == nil {
			r[i] = o
			continue
		}

		var i uint16
		if i, e = IdFromHexString(k); e != nil {
			return
		}
		var b []byte
		if s, ok := v.(string); !ok {
			e = errors.New("invalid parameter for " + k)
			return
		} else if b, e = hex.DecodeString(s); e != nil {
			e = errors.New("invalid parameter for " + k)
			return
		}
		r[i] = b
	}
	*p = r
	return
}

var EncodeParameter func(string, any) (uint16, []byte, error) = nil

func (p *OptionalParameters) readFrom(buf *bytes.Buffer) (e error) {
	r := map[uint16][]byte{}
	for {
		var i, l uint16
		if e = binary.Read(buf, binary.BigEndian, &i); e == io.EOF {
			e = nil
			break
		} else if e != nil {
			return
		} else if e = binary.Read(buf, binary.BigEndian, &l); e != nil {
			return
		}
		v := make([]byte, int(l))
		if _, e = buf.Read(v); e != nil {
			return
		}
		r[i] = v
	}
	*p = r
	return
}

func (p OptionalParameters) writeTo(w *bytes.Buffer) {
	for i, o := range p {
		binary.Write(w, binary.BigEndian, i)
		binary.Write(w, binary.BigEndian, uint16(len(o)))
		w.Write(o)
	}
}

func IdToHexString(i uint16) string {
	return hex.EncodeToString([]byte{byte(i >> 8), byte(i)})
}

func IdFromHexString(s string) (uint16, error) {
	b, e := hex.DecodeString(s)
	if e != nil {
		return 0, fmt.Errorf("invalid id of parameter: %s", e)
	}
	if len(b) != 2 {
		return 0, fmt.Errorf("invalid size of id")
	}
	return (uint16(b[0]) << 8) | uint16(b[1]), nil
}
