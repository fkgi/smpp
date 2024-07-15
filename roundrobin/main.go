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
	fmt.Printf("Usage: %s [OPTION]... ADDRESS\n", os.Args[0])
	fmt.Println("ADDRESS: SMPP peer address when act as ESME, or SMPP local address when act as SMSC. Format is ADDRESS:PORT (default :2775).")
	fmt.Println()
	flag.PrintDefaults()
}

var backend string

func main() {
	var e error

	if smpp.ID, e = os.Hostname(); e != nil {
		smpp.ID = "roundrobin"
	}
	id := flag.String("id", smpp.ID,
		"System ID")
	//ls := flag.String("smpp", "localhost:2775",
	//	"SMPP peer/local address")
	lh := flag.String("http", ":8080",
		"HTTP local address")
	ph := flag.String("backend", "backend:8080",
		"HTTP backend address")
	bt := flag.String("bind", "svr",
		"Bind type of client [tx/rx/trx] or server [svr]")
	pw := flag.String("pwd", "",
		"Password for ESME authentication")
	st := flag.String("type", "",
		"Type of ESME system")
	tn := flag.Int("ton", 0,
		"Type of Number for ESME address")
	np := flag.Int("npi", 0,
		"Numbering Plan Indicator for ESME address")
	ar := flag.String("addr", "",
		"UNIX Regular Expression notation of ESME address")
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

	log.Println("booting Round-Robin debugger for SMPP...")
	if info.BindType == smpp.NilBind {
		log.Println("running as SMSC")
		log.Println("system ID:", *id)
	} else {
		log.Println("running as ESME")
		log.Println("system ID  :", *id)
		log.Println("password   :", *pw)
		log.Println("system type:", *st)
		log.Printf("address    : %s(ton=%d, npi=%d)", *ar, *tn, *np)
	}

	var b smpp.Bind
	sigc := make(chan os.Signal, 1)
	smpp.ConnectionDownNotify = func(_ *smpp.Bind) {
		sigc <- nil
	}
	smpp.RxMessageNotify = func(id smpp.CommandID, stat, seq uint32, body []byte) {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "Rx message")
		fmt.Fprintln(buf, "| command_id     :", id)
		fmt.Fprintln(buf, "| command_status :", stat)
		fmt.Fprintln(buf, "| sequence_number:", seq)
		log.Print(buf)
	}
	smpp.TxMessageNotify = func(id smpp.CommandID, stat, seq uint32, body []byte) {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "Tx message")
		fmt.Fprintln(buf, "| command_id     :", id)
		fmt.Fprintln(buf, "| command_status :", stat)
		fmt.Fprintln(buf, "| sequence_number:", seq)
		log.Print(buf)
	}
	smpp.RequestHandler = handleSMPP

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
	if e != nil {
		log.Println("invalid HTTP backend host, Rx request will be rejected")
		backend = ""
	} else {
		log.Println("HTTP backend is", backend)
	}

	log.Println("local HTTP API port is", *lh)
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
		log.Println("bind is up")
		log.Println("ESME system ID  :", b.PeerID)
		log.Println("ESME password   :", b.Password)
		log.Println("ESME system type:", b.SystemType)
		log.Printf("ESME address    : %s(ton=%d, npi=%d)",
			b.AddressRange, b.TypeOfNumber, b.NumberingPlan)
	} else {
		// run as ESME
		log.Println("connecting SMPP to", addr)
		if c, e := net.Dial("tcp", addr); e != nil {
			log.Fatalln(e)
		} else if b, e = smpp.Connect(c, info); e != nil {
			log.Fatalln(e)
		}
		log.Println("bind is up")
		log.Println("SMSC system ID:", b.PeerID)
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

func handleHTTP(w http.ResponseWriter, r *http.Request, d smpp.Request, b smpp.Bind) {
	if r.Method != http.MethodPost {
		w.Header().Add("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	jsondata, e := io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if e = json.Unmarshal(jsondata, &d); e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	res, e := b.Send(d)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	jsondata, e = json.Marshal(res)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsondata)

}

func handleSMPP(info smpp.BindInfo, req smpp.Request) (stat uint32, res smpp.Response) {
	jsondata, e := json.Marshal(req)
	if e != nil {
		stat = 0x00000008
		return
	}
	var path string
	switch req.(type) {
	case *smpp.DataSM:
		path = "/data"
		res = &smpp.DataSM_resp{}
	case *smpp.DeliverSM:
		path = "/deliver"
		res = &smpp.DeliverSM_resp{}
	case *smpp.SubmitSM:
		path = "/submit"
		res = &smpp.SubmitSM_resp{}
	default:
		stat = 0x00000008
		return
	}

	r, e := http.Post(backend+path, "application/json", bytes.NewBuffer(jsondata))
	if e != nil {
		res = nil
		stat = 0x00000008
		return
	}

	jsondata, e = io.ReadAll(r.Body)
	defer r.Body.Close()
	if e != nil {
		res = nil
		stat = 0x00000008
		return
	}
	if e = json.Unmarshal(jsondata, res); e != nil {
		res = nil
		stat = 0x00000008
		return
	}

	stat = 0
	return
}
