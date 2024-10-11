package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fkgi/smpp"
)

func printHelp() {
	fmt.Printf("Usage: %s [OPTION]... [IP]:PORT\n", os.Args[0])
	fmt.Println("Args")
	fmt.Println("\tIP: SMPP peer address when act as ESME, or SMPP local address when act as SMSC.")
	fmt.Println("\tPORT: SMPP peer port when act as ESME, or SMPP local port when act as SMSC. (default :2775)")
	fmt.Println()
	flag.PrintDefaults()
}

var backend string

func main() {
	var e error

	if smpp.ID, e = os.Hostname(); e != nil {
		smpp.ID = "roundrobin"
	}
	id := flag.String("s", smpp.ID, "System ID")
	lh := flag.String("i", ":8080", "HTTP local address")
	ph := flag.String("b", "", "HTTP backend address")
	bt := flag.String("d", "svr", "Bind type of client [tx/rx/trx] or server [svr]")
	pw := flag.String("p", "", "Password for ESME authentication")
	st := flag.String("y", "DEBUGGER", "Type of ESME system")
	tn := flag.Int("o", 0, "Type of Number for ESME address")
	np := flag.Int("n", 0, "Numbering Plan Indicator for ESME address")
	ar := flag.String("a", "", "UNIX Regular Expression notation of ESME address")
	ts := flag.Bool("t", false, "enable TLS")
	cr := flag.String("c", "", "TLS crt file")
	ky := flag.String("k", "", "TLS key file")
	help := flag.Bool("h", false, "Print usage")
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	addr := flag.Arg(0)
	if addr == "" {
		addr = ":2775"
	}
	smpp.ID = *id
	info := smpp.BindInfo{
		Password:      *pw,
		SystemType:    *st,
		TypeOfNumber:  byte(*tn),
		NumberingPlan: byte(*np),
		AddressRange:  *ar}
	switch *bt {
	case "tx":
		info.BindType = smpp.TxBind
	case "rx":
		info.BindType = smpp.RxBind
	case "trx":
		info.BindType = smpp.TRxBind
	case "svr":
		info.BindType = smpp.NilBind
	default:
		fmt.Println("invalid value", *bt, "for flag -bind")
		printHelp()
		os.Exit(1)
	}
	if *id == "" {
		fmt.Println("invalid empty system ID in flag -id")
		printHelp()
		os.Exit(1)
	}

	log.Println("[INFO]", "booting Round-Robin diagnostic/debug subsystem for SMPP...")
	if info.BindType == smpp.NilBind {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "running as SMSC")
		fmt.Fprintln(buf, "| system ID:", *id)
		log.Print("[INFO]", buf)
	} else {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "running as ESME")
		fmt.Fprintln(buf, "| system ID  :", *id)
		fmt.Fprintln(buf, "| password   :", *pw)
		fmt.Fprintln(buf, "| system type:", *st)
		fmt.Fprintf(buf, "| address    : %s(ton=%d, npi=%d)", *ar, *tn, *np)
		log.Print("[INFO]", buf)
	}

	var b smpp.Bind
	sigc := make(chan os.Signal, 1)
	smpp.ConnectionDownNotify = func(_ *smpp.Bind) {
		sigc <- nil
	}
	smpp.RxMessageNotify = func(id smpp.CommandID, stat smpp.StatusCode, seq uint32, body []byte) {
		log.Printf("[INFO] Rx SMPP message: %s(status=%d, sequence=%d)", id, stat, seq)
	}
	smpp.TxMessageNotify = func(id smpp.CommandID, stat smpp.StatusCode, seq uint32, body []byte) {
		log.Printf("[INFO] Tx SMPP message: %s(status=%d, sequence=%d)", id, stat, seq)
	}

	http.HandleFunc("/smppmsg/v1/data", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.DataSM{}, b)
	})
	http.HandleFunc("/smppmsg/v1/deliver", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.DeliverSM{}, b)
	})
	http.HandleFunc("/smppmsg/v1/submit", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.SubmitSM{}, b)
	})

	backend = "http://" + *ph
	_, e = url.Parse(backend)
	if e != nil || len(*ph) == 0 {
		log.Println("[ERR]", "invalid HTTP backend host, SMPP answer will be always ACK")
		backend = ""
	} else {
		log.Println("[INFO]", "HTTP backend is", backend)
		smpp.RequestHandler = handleSMPP
	}

	log.Println("[INFO]", "local HTTP API address is", *lh)
	go func() {
		e := http.ListenAndServe(*lh, nil)
		if e != nil {
			log.Println("[ERR]", "failed to listen HTTP, Tx request is not available:", e)
		}
	}()

	if info.BindType == smpp.NilBind {
		// run as SMSC
		var l net.Listener
		var e error
		if *ts {
			buf := new(strings.Builder)
			fmt.Fprintln(buf, "listening SMPP on", addr, "with TLS")
			fmt.Fprintln(buf, "| Cert file:", *cr)
			fmt.Fprintln(buf, "| Key file :", *ky)
			log.Print("[INFO]", buf)

			var cer tls.Certificate
			cer, e = tls.LoadX509KeyPair(*cr, *ky)
			if e != nil {
				log.Fatalln(e)
			}
			l, e = tls.Listen("tcp", addr, &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cer}})
		} else {
			log.Println("[INFO]", "listening SMPP on", addr, "without TLS")
			l, e = net.Listen("tcp", addr)
		}
		if e != nil {
			log.Fatalln(e)
		}
		c, e := l.Accept()
		if e != nil {
			log.Fatalln(e)
		}
		l.Close()
		log.Println("[INFO]", "accepting SMPP from", c.RemoteAddr())
		if b, e = smpp.Accept(c); e != nil {
			log.Fatalln(e)
		}

		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is up")
		fmt.Fprintln(buf, "| ESME system ID  :", b.PeerID)
		fmt.Fprintln(buf, "| ESME password   :", b.Password)
		fmt.Fprintln(buf, "| ESME system type:", b.SystemType)
		fmt.Fprintf(buf, "| ESME address    : %s(ton=%d, npi=%d)",
			b.AddressRange, b.TypeOfNumber, b.NumberingPlan)
		log.Println("[INFO]", buf)
	} else {
		// run as ESME
		var c net.Conn
		var e error
		if *ts {
			log.Println("[INFO]", "connecting SMPP to", addr, "with TLS")
			c, e = tls.Dial("tcp", addr, &tls.Config{
				InsecureSkipVerify: true})
		} else {
			log.Println("[INFO]", "connecting SMPP to", addr, "without TLS")
			c, e = net.Dial("tcp", addr)
		}
		if e != nil {
			log.Fatalln(e)
		}
		if b, e = smpp.Connect(c, info); e != nil {
			log.Fatalln(e)
		}

		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is up")
		fmt.Fprintln(buf, "| SMSC system ID:", b.PeerID)
		log.Print("[INFO]", buf)
	}

	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	call := <-sigc
	if call != nil {
		log.Println("[INFO]", "closing bind")
		b.Close()
	}
	log.Println("[INFO]", "bind is down")
	log.Println("[INFO]", "exit...")
}

func handleHTTP(w http.ResponseWriter, r *http.Request, req smpp.Request, b smpp.Bind) {
	if r.Method != http.MethodPost {
		log.Println("[NOTIF]", "invalid HTTP request method", r.Method)
		w.Header().Add("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	jsondata, e := io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("[ERR]", "failed to read HTTP request", e)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if e = json.Unmarshal(jsondata, &req); e != nil {
		log.Println("[ERR]", "failed to unmarshal JSON HTTP request", e)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("[INFO]", "Tx", req)
	res, e := b.Send(req)
	if e != nil {
		log.Println("[ERR]", "failed to send SMPP request", e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("[INFO]", "Rx", res)
	jsondata, e = json.Marshal(res)
	if e != nil {
		log.Println("[ERR]", "failed to marshal JSON SMPP response", e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsondata)
}

func handleSMPP(info smpp.BindInfo, req smpp.Request) (res smpp.Response) {
	log.Println("[INFO]", "Rx", req)

	var path string
	switch req.(type) {
	case *smpp.DataSM:
		path = "/smppmsg/v1/data"
		res = &smpp.DataSM_resp{Status: smpp.StatSysErr}
	case *smpp.DeliverSM:
		path = "/smppmsg/v1/deliver"
		res = &smpp.DeliverSM_resp{Status: smpp.StatSysErr}
	case *smpp.SubmitSM:
		path = "/smppmsg/v1/submit"
		res = &smpp.SubmitSM_resp{Status: smpp.StatSysErr}
	default:
		log.Println("[ERR]", "unknown SMPP request")
		return
	}

	jsondata, e := json.Marshal(req)
	if e != nil {
		log.Println("[ERR]", "failed to marshal JSON SMPP request", e)
		log.Println("[INFO]", "Tx", res)
		return
	}

	r, e := http.Post(backend+path, "application/json", bytes.NewBuffer(jsondata))
	if e != nil {
		log.Println("[ERR]", "failed to send HTTP request,", e)
		log.Println("[INFO]", "Tx", res)
		return
	}

	jsondata, e = io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("[ERR]", "failed to read HTTP response,", e)
	} else if e = json.Unmarshal(jsondata, res); e != nil {
		log.Println("[ERR]", "failed to unmarshal JSON HTTP response,", e)
	}
	log.Println("[INFO]", "Tx", res)
	return
}
