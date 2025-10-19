package main

import (
	"flag"
	"fmt"
	"log"
	"net"

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

var uplink *smpp.Bind = nil

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
				log.Println("[INFO]", "closed, error=", b.ListenAndServe(c))
				break
			}
		}
	}

	la, _, _ := net.SplitHostPort(l.Addr().String())
	localAddr, e := net.ResolveTCPAddr("tcp", la+":0")
	if e != nil {
		log.Fatalln("[ERROR]", "invalid local address", e)
	}

	binds := make([]*smpp.Bind, len(dsts))

}
