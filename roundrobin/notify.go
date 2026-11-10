package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/fkgi/smpp"
	"github.com/fkgi/smpp/dictionary"
)

func init() {
	dictionary.TraceTxHttpRequest = func(
		p string, tx []byte, c int, rx []byte, e error) {
		traceHTTPHandling("Tx", p, tx, c, rx, e)
	}
	dictionary.TraceRxHttpRequest = func(
		p string, tx []byte, c int, rx []byte, e error) {
		traceHTTPHandling("Rx", p, tx, c, rx, e)
	}

	smpp.TraceRxMessage = func(id smpp.CommandID, st smpp.StatusCode, seq uint32, body []byte, e error) {
		traceMessage("Rx", id, st, seq, body, e)
		count(id, st, true)
	}

	smpp.TraceTxMessage = func(id smpp.CommandID, st smpp.StatusCode, seq uint32, body []byte, e error) {
		traceMessage("Tx", id, st, seq, body, e)
		count(id, st, false)
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
		fmt.Fprint(buf, "| peer system ID  :", i.PeerID)
		log.Println("[INFO]", buf)
	}

	dictionary.NotifyHandlerError = func(proto, msg string) {
		log.Println("[ERROR]", "error in", proto, "with reason", msg)
	}
}

func traceMessage(direction string, id smpp.CommandID, st smpp.StatusCode, seq uint32, body []byte, e error) {
	stat := ""
	if !id.IsRequest() {
		stat = fmt.Sprintf(", stat=%s", st)
	}
	if pdu := smpp.MakePDUof(id); pdu == nil {
		log.Printf("[INFO] %s %s (seq=%d%s), error=%v",
			direction, id, seq, stat, e)
	} else if e2 := pdu.Unmarshal(body); e2 != nil {
		log.Printf("[INFO] %s %s (seq=%d%s), error=%v\n| invalid body data: %s",
			direction, id, seq, stat, e, e2)
	} else {
		log.Printf("[INFO] %s %s (seq=%d%s), error=%v%s",
			direction, id, seq, stat, e, pdu)
	}
}

func traceHTTPHandling(d string, p string, tx []byte, c int, rx []byte, e error) {
	buf := new(strings.Builder)
	fmt.Fprintf(buf, "%s HTTP request handling: error=%v\n", d, e)
	fmt.Fprintln(buf, "| path:  ", p)
	fmt.Fprintln(buf, "| body:  ", strings.TrimSpace(string(tx)))
	fmt.Fprintln(buf, "| result:", c, http.StatusText(c))
	fmt.Fprintln(buf, "| body:  ", strings.TrimSpace(string(rx)))
	log.Print("[INFO] ", buf)
}
