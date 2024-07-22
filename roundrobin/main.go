package main

import (
	"bytes"
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
	id := flag.String("id", smpp.ID, "System ID")
	lh := flag.String("http", ":8080", "HTTP local address")
	ph := flag.String("backend", "", "HTTP backend address")
	bt := flag.String("bind", "svr", "Bind type of client [tx/rx/trx] or server [svr]")
	pw := flag.String("pwd", "", "Password for ESME authentication")
	st := flag.String("type", "DEBUGGER", "Type of ESME system")
	tn := flag.Int("ton", 0, "Type of Number for ESME address")
	np := flag.Int("npi", 0, "Numbering Plan Indicator for ESME address")
	ar := flag.String("addr", "", "UNIX Regular Expression notation of ESME address")
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

	log.Println("booting Round-Robin diagnostic/debug subsystem for SMPP...")
	if info.BindType == smpp.NilBind {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "running as SMSC")
		fmt.Fprintln(buf, "| system ID:", *id)
		log.Print(buf)
	} else {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "running as ESME")
		fmt.Fprintln(buf, "| system ID  :", *id)
		fmt.Fprintln(buf, "| password   :", *pw)
		fmt.Fprintln(buf, "| system type:", *st)
		fmt.Fprintf(buf, "| address    : %s(ton=%d, npi=%d)", *ar, *tn, *np)
		log.Print(buf)
	}

	var b smpp.Bind
	sigc := make(chan os.Signal, 1)
	smpp.ConnectionDownNotify = func(_ *smpp.Bind) {
		sigc <- nil
	}
	smpp.RxMessageNotify = func(id smpp.CommandID, stat smpp.StatusCode, seq uint32, body []byte) {
		log.Printf("Rx SMPP message: %s(status=%d, sequence=%d)", id, stat, seq)
	}
	smpp.TxMessageNotify = func(id smpp.CommandID, stat smpp.StatusCode, seq uint32, body []byte) {
		log.Printf("Tx SMPP message: %s(status=%d, sequence=%d)", id, stat, seq)
	}

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.DataSM{}, b)
	})
	http.HandleFunc("/deliver", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.DeliverSM{}, b)
	})
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.SubmitSM{}, b)
	})

	backend = "http://" + *ph
	_, e = url.Parse(backend)
	if e != nil || len(*ph) == 0 {
		log.Println("invalid HTTP backend host, SMPP answer will be always ACK")
		backend = ""
	} else {
		log.Println("HTTP backend is", backend)
		smpp.RequestHandler = handleSMPP
	}

	log.Println("local HTTP API address is", *lh)
	go func() {
		e := http.ListenAndServe(*lh, nil)
		if e != nil {
			log.Println("failed to listen HTTP, Tx request is not available:", e)
		}
	}()

	if info.BindType == smpp.NilBind {
		// run as SMSC
		log.Println("listening SMPP on", addr)
		if l, e := net.Listen("tcp", addr); e != nil {
			log.Fatalln(e)
		} else if c, e := l.Accept(); e != nil {
			log.Fatalln(e)
		} else {
			l.Close()
			log.Println("accepting SMPP from", c.RemoteAddr())
			if b, e = smpp.Accept(c); e != nil {
				log.Fatalln(e)
			}
		}
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is up")
		fmt.Fprintln(buf, "| ESME system ID  :", b.PeerID)
		fmt.Fprintln(buf, "| ESME password   :", b.Password)
		fmt.Fprintln(buf, "| ESME system type:", b.SystemType)
		fmt.Fprintf(buf, "| ESME address    : %s(ton=%d, npi=%d)",
			b.AddressRange, b.TypeOfNumber, b.NumberingPlan)
		log.Print(buf)

		log.Println("bind is up")
	} else {
		// run as ESME
		log.Println("connecting SMPP to", addr)
		if c, e := net.Dial("tcp", addr); e != nil {
			log.Fatalln(e)
		} else if b, e = smpp.Connect(c, info); e != nil {
			log.Fatalln(e)
		}
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is up")
		fmt.Fprintln(buf, "| SMSC system ID:", b.PeerID)
		log.Print(buf)
	}

	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	call := <-sigc
	if call != nil {
		log.Println("closing bind")
		b.Close()
	}
	log.Println("bind is down")
	log.Println("exit...")
}

func handleHTTP(w http.ResponseWriter, r *http.Request, req smpp.Request, b smpp.Bind) {
	if r.Method != http.MethodPost {
		log.Println("invalid HTTP request method", r.Method)
		w.Header().Add("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	jsondata, e := io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("failed to read HTTP request", e)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if e = json.Unmarshal(jsondata, &req); e != nil {
		log.Println("failed to unmarshal JSON HTTP request", e)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("Tx", req)
	res, e := b.Send(req)
	if e != nil {
		log.Println("failed to send SMPP request", e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("Rx", res)
	jsondata, e = json.Marshal(res)
	if e != nil {
		log.Println("failed to marshal JSON SMPP response", e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsondata)
}

func handleSMPP(info smpp.BindInfo, req smpp.Request) (res smpp.Response) {
	log.Println("Rx", req)

	var path string
	switch req.(type) {
	case *smpp.DataSM:
		path = "/data"
		res = &smpp.DataSM_resp{Status: smpp.StatSysErr}
	case *smpp.DeliverSM:
		path = "/deliver"
		res = &smpp.DeliverSM_resp{Status: smpp.StatSysErr}
	case *smpp.SubmitSM:
		path = "/submit"
		res = &smpp.SubmitSM_resp{Status: smpp.StatSysErr}
	default:
		log.Println("unknown SMPP request")
		return
	}

	jsondata, e := json.Marshal(req)
	if e != nil {
		log.Println("failed to marshal JSON SMPP request", e)
		log.Println("Tx", res)
		return
	}

	r, e := http.Post(backend+path, "application/json", bytes.NewBuffer(jsondata))
	if e != nil {
		log.Println("failed to send HTTP request,", e)
		log.Println("Tx", res)
		return
	}

	jsondata, e = io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		log.Println("failed to read HTTP response,", e)
	} else if e = json.Unmarshal(jsondata, res); e != nil {
		log.Println("failed to unmarshal JSON HTTP response,", e)
	}
	log.Println("Tx", res)
	return
}
