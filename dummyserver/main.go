package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"time"
	"unicode/utf16"
)

func main() {
	log.Println("Starting dummy SMPP server")
	addr := flag.String("l", ":2775", "listen address")
	flag.Parse()

	l, e := net.Listen("tcp", *addr)
	if e != nil {
		log.Fatalln(e)
	}
	c, e := l.Accept()
	if e != nil {
		log.Fatalln(e)
	}
	defer c.Close()

	mid, _, num, _, e := readPDU(c)
	if e != nil {
		log.Fatalln(e)
	}
	if mid != 0x00000002 {
		log.Fatalln("invalid request")
	}
	mid |= 0x80000000

	w := new(bytes.Buffer)
	w.WriteString("DUMMY")
	w.WriteByte(0)
	// interface_version
	binary.Write(w, binary.BigEndian, uint16(0x0210))
	binary.Write(w, binary.BigEndian, uint16(1))
	w.WriteByte(0x34)

	e = writePDU(c, mid, 0, num, w.Bytes())
	if e != nil {
		return
	}

	counterd := 0
	counters := 0

	for {
		var body []byte
		var cause uint32
		mid, _, num, body, e = readPDU(c)
		if e != nil {
			log.Fatalln(e)
		}
		switch mid {
		case 0x00000015: // enquire_link
			body, e = nil, nil
		case 0x00000006: // unbind
			body, e = nil, nil
		case 0x00000004: // submit_sm
			switch counters {
			case 1:
				cause = 0x000000FE
				body, e = nil, nil
			case 2:
				cause = 0x00000058
				body, e = nil, nil
			case 4:
				e = writePDU(c, 0x80000000, 0x00000003, num, nil)
				if e != nil {
					log.Fatalln(e)
				}
				e = errors.New("skip")
			case 3:
				e = errors.New("skip")
			default:
				w := new(bytes.Buffer)
				w.WriteString(time.Now().String())
				w.WriteByte(0)
				body = w.Bytes()
				e = nil
			}
			counters++
		case 0x00000103: // data_sm
			switch counterd {
			case 1:
				cause = 0x000000FE
				body, e = nil, nil
			case 2:
				cause = 0x00000058
				body, e = nil, nil
			case 3:
				u := utf16.Encode([]rune("てすと"))
				ud := make([]byte, len(u)*2)
				for i, c := range u {
					ud[i*2] = byte((c >> 8) & 0xff)
					ud[i*2+1] = byte(c & 0xff)
				}

				w := new(bytes.Buffer)
				w.WriteString(time.Now().String())
				w.WriteByte(0)
				binary.Write(w, binary.BigEndian, uint16(0x0424))
				binary.Write(w, binary.BigEndian, uint16(len(ud)))
				w.Write(ud)
				body = w.Bytes()
				e = nil
			case 4:
				e = writePDU(c, 0x80000000, 0x00000003, num, nil)
				if e != nil {
					log.Fatalln(e)
				}
				e = errors.New("skip")
			case 5:
				e = errors.New("skip")
			default:
				w := new(bytes.Buffer)
				w.WriteString(time.Now().String())
				w.WriteByte(0)
				body = w.Bytes()
				e = nil
			}
			counterd++
		}
		if e != nil {
			continue
		}
		mid |= 0x80000000
		e = writePDU(c, mid, cause, num, body)
		if e != nil {
			log.Fatalln(e)
		}

		if mid == 0x80000006 {
			time.Sleep(time.Second)
			break
		}
	}
}

func readPDU(r io.Reader) (id, stat, num uint32, body []byte, e error) {
	var l uint32
	if e = binary.Read(r, binary.BigEndian, &l); e != nil {
		return
	}
	if l < 16 {
		e = errors.New("invalid header")
		return
	}
	l -= 16
	if e = binary.Read(r, binary.BigEndian, &id); e != nil {
		return
	}
	if e = binary.Read(r, binary.BigEndian, &stat); e != nil {
		return
	}
	if e = binary.Read(r, binary.BigEndian, &num); e != nil {
		return
	}
	if l != 0 {
		b := make([]byte, l)
		offset := 0
		n := 0
		for offset < 1 {
			n, e = r.Read(b[offset:])
			offset += n
			if e != nil {
				break
			}
		}
	}
	return
}

func writePDU(w io.Writer, id, stat, num uint32, body []byte) (e error) {
	if body == nil {
		body = []byte{}
	}
	buf := bufio.NewWriter(w)

	// command_length
	binary.Write(buf, binary.BigEndian, uint32(len(body)+16))
	// command_id
	binary.Write(buf, binary.BigEndian, id)
	// command_status
	binary.Write(buf, binary.BigEndian, stat)
	// sequence_number
	binary.Write(buf, binary.BigEndian, num)

	buf.Write(body)
	return buf.Flush()
}
