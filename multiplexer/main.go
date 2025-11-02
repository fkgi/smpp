package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/fkgi/smpp"
)

type destAddrs []string

func (a *destAddrs) String() string {
	return fmt.Sprintf("%v", *a)
}
func (a *destAddrs) Set(v string) error {
	*a = append(*a, v)
	return nil
}

func main() {
	var dst destAddrs
	flag.Var(&dst, "r", "SMPP destination address and port")
	src := flag.String("l", "", "SMPP local address and port")

	flag.Parse()

	dsts := make([]*net.TCPAddr, 0, len(dst))
	for _, d := range dst {
		if a, e := net.ResolveTCPAddr("tcp", d); e != nil {
			log.Fatalln("[ERROR]", "invalid destination address:", e)
		} else {
			dsts = append(dsts, a)
		}
	}

	localAddr, e := net.ResolveTCPAddr("tcp", *src)
	if e != nil {
		log.Fatalln("[ERROR]", "invalid local address", e)
	}
	localAddr.Port = 0

	var uplink *smpp.Bind = nil
	var downLink = make([]*smpp.Bind, len(dsts))

	for i := 0; i < len(downLink); i++ {
		go func(i int, a *net.TCPAddr) {
			for {
				time.Sleep(time.Second * 30)
				if downLink[i] == nil {
					continue
				}
				log.Println("[INFO]", "connecting bind to", a)
				c, e := net.DialTCP("tcp", localAddr, a)
				if e != nil {
					continue
				}
				log.Println("[ERROR}", downLink[i].DialAndServe(c))
				downLink[i] = nil
			}
		}(i, dsts[i])
	}

	smpp.BoundNotify = func(bi smpp.BindInfo, a net.Addr) {
		log.Println("[INFO]", "bind is up",
			"\n| peer address    :", a,
			"\n| peer system ID  :", bi.PeerID,
			"\n| bind type       :", bi.BindType)
		if bi.Metadata != "uplink" {
			return
		}

		bi.Metadata = ""
		for i := 0; i < len(dsts); i++ {
			downLink[i] = &smpp.Bind{BindInfo: bi}
		}
	}

	smpp.UnboundNotify = func(i smpp.BindInfo, a net.Addr) {
		log.Println("[INFO]", "bind is down",
			"\n| peer address    :", a,
			"\n| peer system ID  :", i.PeerID)
	}

	smpp.RequestHandler = func(bi smpp.BindInfo, pdu smpp.PDU) (smpp.StatusCode, smpp.PDU) {
		var dest *smpp.Bind
		if bi.Metadata == "uplink" {
			offset := int(rand.Uint32()) % len(downLink)
			for i := offset + 1; dest == nil; i++ {
				if i == len(downLink) {
					i = 0
				}
				if i == offset {
					break
				}
				if downLink[i] != nil && downLink[i].IsActive() {
					dest = downLink[i]
				}
			}
		} else {
			dest = uplink
		}

		if dest == nil {
			return smpp.StatSysErr, smpp.MakePDUof(smpp.GenericNack)
		}

		s, a, e := dest.Send(pdu)
		if e != nil {
			s = smpp.StatSysErr
			a = smpp.MakePDUof(smpp.GenericNack)
		}
		return s, a
	}

	for {
		uplink = nil
		l, e := net.Listen("tcp", *src)
		if e != nil {
			log.Fatalln("[ERROR]", "failed to listen on", *src, ":", e)
		}

		log.Println("[INFO]", "listening SMPP on", l.Addr())
		for {
			if c, e := l.Accept(); e != nil {
				log.Println("[ERROR]", "failed to accept SMPP connection:", e)
			} else {
				l.Close()
				b := &smpp.Bind{}
				b.Metadata = "uplink"
				log.Println("[INFO]", "closed, error=", b.ListenAndServe(c))
				break
			}
		}

		for i := 0; i < len(downLink); i++ {
			if downLink[i] != nil && downLink[i].IsActive() {
				downLink[i].Close()
				downLink[i] = nil
			}
		}
	}
}
