package dictionary

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
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
	encParams = make(map[string]func(any) (uint16, smpp.OctetData, error))
	decParams = make(map[uint16]func(smpp.OctetData) (string, any, error))
)

func LoadDictionary(data []byte) (xd XDictionary, e error) {
	if e = xml.Unmarshal(data, &xd); e != nil {
		return xd, e
	}
	for _, p := range xd.P {
		var i []byte
		if i, e = hex.DecodeString(p.I); e != nil {
			e = fmt.Errorf("invalid id of parameter: %s", e)
			return
		} else if len(i) != 2 {
			e = fmt.Errorf("invalid size of id")
			return
		}
		var id uint16
		if e = binary.Read(bytes.NewBuffer(i), binary.BigEndian, &id); e != nil {
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
			encParams[p.N] = func(v any) (uint16, smpp.OctetData, error) {
				i, ok := v.(string)
				if !ok {
					return id, nil, errors.New("data type mismatch")
				}
				return id, append([]byte(i), 0x00), nil
			}
			decParams[id] = func(d smpp.OctetData) (string, any, error) {
				if len(d) == 0 {
					return n, "", errors.New("data type mismatch")
				}
				return n, string(d[:len(d)-1]), nil
			}
		case "OctetString":
			encParams[p.N] = func(v any) (uint16, smpp.OctetData, error) {
				i, ok := v.([]byte)
				if !ok {
					return id, nil, errors.New("data type mismatch")
				}
				return id, smpp.OctetData(i), nil
			}
			decParams[id] = func(d smpp.OctetData) (string, any, error) {
				return n, []byte(d), nil
			}
		case "Enumerated":
			encEnum := map[string]byte{}
			decEnum := map[byte]string{}
			for _, v := range p.E {
				encEnum[v.V] = v.I
				decEnum[v.I] = v.V
			}
			encParams[p.N] = func(v any) (uint16, smpp.OctetData, error) {
				i, ok := v.(string)
				if !ok {
					return id, nil, errors.New("data type mismatch")
				}
				b, ok := encEnum[i]
				if !ok {
					return id, nil, errors.New("invalid enum data")
				}
				return id, smpp.OctetData{b}, nil
			}
			decParams[id] = func(d smpp.OctetData) (string, any, error) {
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
			encParams[p.N] = func(v any) (uint16, smpp.OctetData, error) {
				return id, smpp.OctetData{}, nil
			}
			decParams[id] = func(d smpp.OctetData) (string, any, error) {
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
	encParams[n] = func(v any) (uint16, smpp.OctetData, error) {
		i, ok := v.(float64)
		if !ok {
			return id, nil, errors.New("data type mismatch")
		}
		w := new(bytes.Buffer)
		binary.Write(w, binary.BigEndian, i)
		b := w.Bytes()
		return id, smpp.OctetData(b[len(b)-l:]), nil
	}
	decParams[id] = func(d smpp.OctetData) (string, any, error) {
		if len(d) != l {
			return n, 0, errors.New("data type mismatch")
		}
		var v uint64
		e := binary.Read(bytes.NewBuffer(d), binary.BigEndian, &v)
		return n, v, e
	}
}

func init() {
	smpp.DecodeParameter = func(i uint16, o smpp.OctetData) (string, any, error) {
		if f, ok := decParams[i]; ok {
			return f(o)
		}
		return "", nil, errors.New("unknown parameter")
	}

	smpp.EncodeParameter = func(s string, a any) (uint16, smpp.OctetData, error) {
		if f, ok := encParams[s]; ok {
			return f(a)
		}
		return 0, nil, errors.New("unknown parameter")
	}
}
