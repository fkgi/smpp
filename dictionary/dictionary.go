package dictionary

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/fkgi/smpp"
)

type XDictionary struct {
	XMLName xml.Name `xml:"dictionary"`
	P       []struct {
		N string `xml:"name,attr"`
		I string `xml:"id,attr"`
		T string `xml:"type,attr"`
		E []struct {
			I byte   `xml:"value,attr"`
			V string `xml:",chardata"`
		} `xml:"enum"`
	} `xml:"parameter"`
}

var (
	encParams = make(map[string]func(any) (uint16, []byte, error))
	decParams = make(map[uint16]func([]byte) (string, any, error))
)

func LoadDictionary(data []byte) (xd XDictionary, e error) {
	if e = xml.Unmarshal(data, &xd); e != nil {
		return xd, e
	}
	for _, p := range xd.P {
		var id uint16
		if id, e = smpp.IdFromHexString(p.I); e != nil {
			return
		}
		n := p.N

		switch p.T {
		case "Integer":
			regIntegerFunc(id, p.N, 1)
		case "Integer2":
			regIntegerFunc(id, p.N, 2)
		case "Integer3":
			regIntegerFunc(id, p.N, 3)
		case "Integer4":
			regIntegerFunc(id, p.N, 4)
		case "CString":
			encParams[p.N] = func(v any) (uint16, []byte, error) {
				i, ok := v.(string)
				if !ok {
					return id, nil, errors.New("data type mismatch")
				}
				return id, append([]byte(i), 0x00), nil
			}
			decParams[id] = func(d []byte) (string, any, error) {
				if len(d) == 0 {
					return n, "", errors.New("data type mismatch")
				}
				return n, string(d[:len(d)-1]), nil
			}
		case "OctetString":
			encParams[p.N] = func(v any) (uint16, []byte, error) {
				i, ok := v.([]byte)
				if !ok {
					return id, nil, errors.New("data type mismatch")
				}
				return id, i, nil
			}
			decParams[id] = func(d []byte) (string, any, error) {
				return n, d, nil
			}
		case "Enumerated":
			encEnum := map[string]byte{}
			decEnum := map[byte]string{}
			for _, v := range p.E {
				encEnum[v.V] = v.I
				decEnum[v.I] = v.V
			}
			encParams[p.N] = func(v any) (uint16, []byte, error) {
				i, ok := v.(string)
				if !ok {
					return id, nil, errors.New("data type mismatch")
				}
				b, ok := encEnum[i]
				if !ok {
					return id, nil, errors.New("invalid enum data")
				}
				return id, []byte{b}, nil
			}
			decParams[id] = func(d []byte) (string, any, error) {
				if len(d) != 1 {
					return n, 0, errors.New("data type mismatch")
				}
				v, ok := decEnum[d[0]]
				if !ok {
					return n, "", errors.New("invalid enum data")
				}
				return n, v, nil
			}
		case "Null":
			encParams[p.N] = func(v any) (uint16, []byte, error) {
				return id, []byte{}, nil
			}
			decParams[id] = func(d []byte) (string, any, error) {
				if len(d) != 0 {
					return n, nil, errors.New("data type mismatch")
				}
				return n, nil, nil
			}
		}
	}
	return
}

func regIntegerFunc(id uint16, n string, l int) {
	encParams[n] = func(v any) (uint16, []byte, error) {
		var u uint64
		if f, ok := v.(float64); !ok {
			return id, nil, errors.New("data type mismatch")
		} else {
			u = uint64(f)
		}
		d := make([]byte, l)
		for i := range d {
			d[i] = byte(u >> (8 * (l - i - 1)))
		}
		return id, smpp.OctetData(d), nil
	}
	decParams[id] = func(d []byte) (string, any, error) {
		if len(d) != l {
			return n, 0, errors.New("data type mismatch")
		}
		var v uint64 = 0
		for _, b := range d {
			v = (v << 8) | uint64(b)
		}
		return n, v, nil
	}
}

func init() {
	smpp.DecodeParameter = func(i uint16, o []byte) (string, any, error) {
		if f, ok := decParams[i]; ok {
			s, a, e := f(o)
			fmt.Println(i, o, s, a, e)
			return s, a, e
		}
		return "", nil, errors.New("unknown parameter")
	}

	smpp.EncodeParameter = func(s string, a any) (uint16, []byte, error) {
		if f, ok := encParams[s]; ok {
			i, o, e := f(a)
			fmt.Println(s, a, i, o, e)
			return i, o, e
		}
		return 0, nil, errors.New("unknown parameter")
	}
}
