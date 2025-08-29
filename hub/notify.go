package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/fkgi/smpp"
	"github.com/fkgi/smpp/dictionary"
)

func init() {
	smpp.TraceMessage = func(di smpp.Direction, id smpp.CommandID, st smpp.StatusCode, seq uint32, body []byte) {
		stat := ""
		if !id.IsRequest() {
			stat = fmt.Sprintf(", stat=%s", st)
		}
		if pdu := smpp.MakePDUof(id); pdu == nil {
			log.Printf("[INFO] %s %s (seq=%d%s)", di, id, seq, stat)
		} else if e := pdu.Unmarshal(body); e != nil {
			log.Printf("[INFO] %s %s (seq=%d%s)\n| error: %s", di, id, seq, stat, e)
		} else {
			log.Printf("[INFO] %s %s (seq=%d%s)%s", di, id, seq, stat, pdu)
		}
	}

	smpp.BoundNotify = func(i smpp.BindInfo, a net.Addr) {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is up")
		fmt.Fprintln(buf, "| peer address    :", a)
		fmt.Fprintln(buf, "| peer system ID  :", i.PeerID)
		fmt.Fprint(buf, "| bind type       : ", i.BindType)
		log.Println("[INFO]", buf)
	}

	smpp.UnboundNotify = func(i smpp.BindInfo, a net.Addr) {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is down")
		fmt.Fprintln(buf, "| peer address    :", a)
		fmt.Fprintln(buf, "| peer system ID  :", i.PeerID)
		log.Println("[INFO]", buf)
	}

	dictionary.NotifyHandlerError = func(proto, msg string) {
		log.Println("[ERROR]", "error in", proto, "with reason", msg)
	}
}
