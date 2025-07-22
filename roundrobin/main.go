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
	backend  string
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
	bindType = flag.String("d", "svr", "Bind type of client [tx/rx/trx] or server [svr]")
	pw := flag.String("p", "", "Password for ESME authentication")
	st := flag.String("y", "ROUNDROBIN", "Type of ESME system")
	tn := flag.Uint("o", 0, "Type of Number for ESME address")
	np := flag.Uint("n", 0, "Numbering Plan Indicator for ESME address")
	ar := flag.String("a", "", "UNIX Regular Expression notation of ESME address")
	ts := flag.Bool("t", false, "enable TLS")
	cr := flag.String("c", "", "TLS crt file")
	ky := flag.String("k", "", "TLS key file")
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

	addr := flag.Arg(0)
	if addr == "" {
		addr = ":2775"
	}

	if *id == "" {
		fmt.Println("invalid empty system ID in flag -s")
		printHelp()
		os.Exit(1)
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
		fmt.Println("invalid bind type", *bindType, "for flag -d")
		printHelp()
		os.Exit(1)
	}
	bind.Password = *pw
	bind.SystemType = *st
	bind.TypeOfNumber = byte(*tn)
	bind.NumberingPlan = byte(*np)
	bind.AddressRange = *ar

	log.Println("[INFO]", "booting Round-Robin diagnostic/debug subsystem for SMPP...")
	if *bindType == "svr" {
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

	sigc := make(chan os.Signal, 1)

	backend = "http://" + *ph
	_, e = url.Parse(backend)
	if e != nil || len(*ph) == 0 {
		log.Println("[ERROR]", "invalid HTTP backend host, SMPP answer will be always failed")
		backend = ""
	} else {
		log.Println("[INFO]", "HTTP backend:", backend)
		smpp.RequestHandler = handleSMPP
	}

	http.HandleFunc("/smppmsg/v1/data", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.DataSM{}, bind)
	})
	http.HandleFunc("/smppmsg/v1/deliver", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.DeliverSM{}, bind)
	})
	http.HandleFunc("/smppmsg/v1/submit", func(w http.ResponseWriter, r *http.Request) {
		handleHTTP(w, r, &smpp.SubmitSM{}, bind)
	})

	log.Println("[INFO]", "listening HTTP...\n| local port:", *lh)
	go func() {
		e := http.ListenAndServe(*lh, nil)
		if e != nil {
			log.Println("[WARN]", "failed to listen HTTP, Tx request is not available:", e)
		}
	}()

	go func() {
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		if call := <-sigc; call != nil {
			log.Println("[INFO]", "closing bind")
			bind.Close()
			time.Sleep(time.Second * 5)
			os.Exit(0)
		}
	}()

	if *bindType == "svr" {
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

		log.Println("[INFO]", "closed, error=", bind.DialAndServe(c))
	}
}
