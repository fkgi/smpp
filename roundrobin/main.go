package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fkgi/smpp"
	"github.com/fkgi/smpp/dictionary"
	"github.com/fkgi/teldata"
)

func printHelp() {
	fmt.Printf("Usage: %s [OPTION]... [[IP]:PORT]\n", os.Args[0])
	fmt.Println("Args")
	fmt.Println("\tIP: SMPP peer address when act as ESME, or SMPP local address when act as SMSC.")
	fmt.Println("\tPORT: SMPP peer port when act as ESME, or SMPP local port when act as SMSC. (default :2775)")
	fmt.Println()
	flag.PrintDefaults()
}

var (
	verbose  *bool
	bindType *string
)

func main() {
	var e error
	if smpp.ID, e = os.Hostname(); e != nil {
		smpp.ID = "roundrobin"
	}
	id := flag.String("s", smpp.ID, "System ID")
	lh := flag.String("i", ":8080", "HTTP local address")
	ph := flag.String("b", "", "HTTP backend address")
	bindType = flag.String("x", "svr", "Bind type of client [tx/rx/trx] or server [svr]")
	pw := flag.String("p", "", "Password for ESME authentication")
	st := flag.String("y", "DEBUGGER", "Type of ESME system")
	tn := flag.Uint("o", 0, "Type of Number for ESME address")
	np := flag.Uint("n", 0, "Numbering Plan Indicator for ESME address")
	ar := flag.String("a", "", "UNIX Regular Expression notation of ESME address")
	ts := flag.Bool("t", false, "enable TLS")
	cr := flag.String("c", "", "TLS crt file")
	ky := flag.String("k", "", "TLS key file")
	dict := flag.String("d", "dictionary.xml", "SMPP dictionary file `path`.")
	help := flag.Bool("h", false, "Print usage")
	verbose = flag.Bool("v", false, "Verbose log output")
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if !*verbose {
		smpp.TraceMessage = nil
	}

	log.Println("[INFO]", "loading dictionary file", *dict)
	if data, e := os.ReadFile(*dict); e != nil {
		log.Fatalln("[ERROR]", "failed to open dictionary file:", e)
	} else if dicData, e := dictionary.LoadDictionary(data); e != nil {
		log.Fatalln("[ERROR]", "failed to read dictionary file:", e)
	} else {
		buf := new(strings.Builder)
		fmt.Fprint(buf, "supported parameter:")
		for _, p := range dicData.P {
			fmt.Fprintf(buf, " %s(%s/%s),", p.N, p.I, p.T)
		}
		log.Println("[INFO]", buf)
	}

	addr := flag.Arg(0)
	if addr == "" {
		addr = ":2775"
	}

	if *id == "" {
		log.Fatalln("[ERROR]", "invalid empty system ID")
	}
	smpp.ID = *id

	bind := smpp.Bind{}
	switch *bindType {
	case "tx":
		bind.BindType = smpp.TxBind
	case "rx":
		bind.BindType = smpp.RxBind
	case "trx":
		bind.BindType = smpp.TRxBind
	case "svr":
		bind.BindType = smpp.NilBind
	default:
		log.Fatalln("[ERROR]", "invalid bind type", *bindType)
	}
	bind.Password = *pw
	bind.SystemType = *st
	bind.TypeOfNumber = teldata.NatureOfAddress(*tn)
	bind.NumberingPlan = teldata.NumberingPlan(*np)
	bind.AddressRange = *ar

	log.Println("[INFO]", "booting Round-Robin diagnostic/debug subsystem for SMPP...")
	if *bindType == "svr" {
		log.Println("[INFO]", "running as SMSC",
			"\n| system ID:", *id)
	} else {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "running as ESME")
		fmt.Fprintln(buf, "| system ID  :", *id)
		fmt.Fprintln(buf, "| password   :", *pw)
		fmt.Fprintln(buf, "| system type:", *st)
		fmt.Fprintf(buf, "| address    : %s(ton=%d, npi=%d)", *ar, *tn, *np)
		log.Print("[INFO]", buf)
	}

	dictionary.Backend = "http://" + *ph
	_, e = url.Parse(dictionary.Backend)
	if e != nil || len(*ph) == 0 {
		log.Println("[ERROR]", "invalid HTTP backend host, SMPP answer will be always failed")
		dictionary.Backend = ""
	} else {
		log.Println("[INFO]", "HTTP backend:", dictionary.Backend)
		smpp.RequestHandler = dictionary.HandleSMPP
	}

	http.HandleFunc("/smppmsg/v1/data", func(w http.ResponseWriter, r *http.Request) {
		dictionary.HandleHTTP(w, r, &smpp.DataSM{}, &bind)
	})
	http.HandleFunc("/smppmsg/v1/deliver", func(w http.ResponseWriter, r *http.Request) {
		dictionary.HandleHTTP(w, r, &smpp.DeliverSM{}, &bind)
	})
	http.HandleFunc("/smppmsg/v1/submit", func(w http.ResponseWriter, r *http.Request) {
		dictionary.HandleHTTP(w, r, &smpp.SubmitSM{}, &bind)
	})

	log.Println("[INFO]", "listening HTTP...\n| local port:", *lh)
	go func() {
		e := http.ListenAndServe(*lh, nil)
		if e != nil {
			log.Println("[WARN]", "failed to listen HTTP, Tx request is not available:", e)
		}
	}()

	close := func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		if call := <-sigc; call != nil {
			log.Println("[INFO]", "closing bind")
			bind.Close()
			time.Sleep(time.Second * 5)
			os.Exit(0)
		}
	}

	if *bindType == "svr" {
		// run as SMSC
		var l net.Listener
		var e error
		if *ts {
			log.Println("[INFO]", "listening SMPP on", addr, "with TLS",
				"\n| Cert file:", *cr,
				"\n| Key file :", *ky)

			var cer tls.Certificate
			if cer, e = tls.LoadX509KeyPair(*cr, *ky); e == nil {
				l, e = tls.Listen("tcp", addr, &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cer}})
			}
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

		go close()
		log.Println("[INFO]", "closed, error=", bind.ListenAndServe(c))
	} else {
		// run as ESME
		var c net.Conn
		var e error
		if *ts {
			log.Println("[INFO]", "connecting SMPP to", addr, "with TLS")
			c, e = tls.Dial("tcp", addr,
				&tls.Config{InsecureSkipVerify: true})
		} else {
			log.Println("[INFO]", "connecting SMPP to", addr, "without TLS")
			c, e = net.Dial("tcp", addr)
		}
		if e != nil {
			log.Fatalln(e)
		}

		go close()
		log.Println("[INFO]", "closed, error=", bind.DialAndServe(c))
	}
}
