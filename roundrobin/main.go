package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fkgi/smpp"
)

func main() {
	var e error

	smpp.ID, e = os.Hostname()
	if e != nil {
		smpp.ID = "roundrobin"
	}
	id := flag.String("i", smpp.ID, "Host ID")
	ls := flag.String("r", "localhost:2775", "SMPP peer Port")
	lh := flag.String("h", ":8080", "HTTP local Port")
	bt := flag.String("bind", "trx", "Bind type tx/rx/trx")
	pw := flag.String("pwd", "password", "Password")
	st := flag.String("type", "TEST", "System type")
	ton := flag.Int("ton", 0, "TON")
	npi := flag.Int("npi", 0, "NPI")
	adr := flag.String("a", "", "Address range")
	flag.Parse()
	smpp.ID = *id

	info := smpp.BindInfo{
		Password:      *pw,
		SystemType:    *st,
		TypeOfNumber:  byte(*ton),
		NumberingPlan: byte(*npi),
		AddressRange:  *adr}
	switch *bt {
	case "tx":
		info.BindType = smpp.TxBind
	case "rx":
		info.BindType = smpp.RxBind
	default:
		info.BindType = smpp.TRxBind
	}
	c, e := net.Dial("tcp", *ls)
	if e != nil {
		log.Fatalln(e)
	}
	b, e := smpp.Connect(c, info)
	if e != nil {
		log.Fatalln(e)
	}

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
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
		d := smpp.DataSM{}
		if e = json.Unmarshal(jsondata, &d); e != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b.Send(&d)
	})
	http.HandleFunc("/deliver", func(w http.ResponseWriter, r *http.Request) {
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
		d := smpp.DeliverSM{}
		if e = json.Unmarshal(jsondata, &d); e != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b.Send(&d)
	})
	go func() {
		log.Println(http.ListenAndServe(*lh, nil))
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sigc
	b.Close()

	/*
		time.Sleep(time.Second)
		c, e := net.Dial("tcp", "localhost:2775")
		if e != nil {
			log.Fatalln(e)
		}
		b, e := smpp.Connect(c,
			smpp.BindInfo{
				BindType:      smpp.TRxBind,
				Password:      "passwod",
				SystemType:    "TEST",
				TypeOfNumber:  0x00,
				NumberingPlan: 0x00,
				AddressRange:  ""})
		if e != nil {
			log.Fatalln(e)
		}

		param := make(map[uint16][]byte)
		param[0x0424] = []byte{0x00, 0x01, 0x02}
		b.Send(&smpp.DataSM{
			SvcType: "svc",
			SrcAddr: "123",
			DstAddr: "987",
			Param:   param,
		})
		time.Sleep(time.Second)

		b.Send(&smpp.DeliverSM{
			SvcType:      "svc",
			SrcAddr:      "123",
			DstAddr:      "987",
			ShortMessage: []byte{0x09, 0x08, 0x07},
		})
		time.Sleep(time.Second * 10)

		b.Close()
		time.Sleep(time.Second)
	*/
}
