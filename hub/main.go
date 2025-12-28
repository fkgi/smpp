package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
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

var (
	frontend  string
	verbose   *bool
	localAddr *net.TCPAddr = nil
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
	src := flag.String("l", "", "SMPP local address")
	var e error
	if smpp.ID, e = os.Hostname(); e != nil {
		smpp.ID = "hub"
	}
	id := flag.String("s", smpp.ID, "System ID")
	lh := flag.String("i", "", "HTTP local address")
	ph := flag.String("b", "", "HTTP backend address")
	pw := flag.String("p", "", "Password for ESME authentication")
	st := flag.String("y", "DEBUGGER", "Type of ESME system")
	tn := flag.Uint("o", 0, "Type of Number for ESME address")
	np := flag.Uint("n", 0, "Numbering Plan Indicator for ESME address")
	ar := flag.String("a", "", "UNIX Regular Expression notation of ESME address")
	dict := flag.String("d", "dictionary.xml", "Diameter dictionary file `path`.")
	gsm := flag.Bool("g", false, "Set SMSC default alphabet to GSM 7bit")
	help := flag.Bool("h", false, "Print usage")
	to := flag.Int("t", int(smpp.Expire/time.Second), "Message timeout timer [s]")
	verbose = flag.Bool("v", false, "Verbose log output")
	flag.Parse()

	if *help {
		fmt.Printf("Usage: %s [OPTION]...\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	if !*verbose {
		smpp.TraceMessage = nil
	}
	smpp.ID = *id
	smpp.Expire = time.Duration(*to) * time.Second

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

	smpp.DefaultAlphabetIsGSM = *gsm

	if len(*lh) != 0 {
		if a, e := net.ResolveTCPAddr("tcp", *lh); e != nil {
			log.Fatalln("[ERROR]", "invalid HTTP local address:", e)
		} else {
			frontend = a.String()
			log.Println("[INFO]", "HTTP interface:", frontend)
		}
	}
	if len(*ph) != 0 {
		dictionary.Backend = "http://" + *ph
		if _, e = url.Parse(dictionary.Backend); e != nil {
			log.Fatalln("[ERROR]", "invalid HTTP backend host:", e)
		}
		log.Println("[INFO]", "HTTP backend:", dictionary.Backend)
		smpp.RequestHandler = dictionary.HandleSMPP
	}

	info := smpp.BindInfo{
		Password:      *pw,
		SystemType:    *st,
		TypeOfNumber:  teldata.NatureOfAddress(*tn),
		NumberingPlan: teldata.NumberingPlan(*np),
		AddressRange:  *ar}
	if len(frontend) != 0 && len(dictionary.Backend) != 0 {
		info.BindType = smpp.TRxBind
	} else if len(frontend) != 0 {
		info.BindType = smpp.TxBind
	} else if len(dictionary.Backend) != 0 {
		info.BindType = smpp.RxBind
	} else {
		log.Fatalln("[ERROR]", "no HTTP side information")
	}

	if *src != "" {
		if localAddr, e = net.ResolveTCPAddr("tcp", *src+":0"); e != nil {
			log.Fatalln("[ERROR]", "invalid local address", e)
		}
	}

	dsts := []*net.TCPAddr{}
	for _, d := range dst {
		if a, e := net.ResolveTCPAddr("tcp", d); e != nil {
			log.Fatalln("[ERROR]", "invalid destination address", e)
		} else {
			dsts = append(dsts, a)
		}
	}
	binds := make([]*smpp.Bind, len(dsts))

	if info.BindType != smpp.RxBind {
		http.HandleFunc("/smppmsg/v1/data", func(w http.ResponseWriter, r *http.Request) {
			handleHTTP(w, r, &smpp.DataSM{}, binds)
		})
		http.HandleFunc("/smppmsg/v1/submit", func(w http.ResponseWriter, r *http.Request) {
			handleHTTP(w, r, &smpp.SubmitSM{}, binds)
		})
	}
	if len(frontend) != 0 {
		log.Println("[INFO]", "listening HTTP...")
		go func() {
			if e := http.ListenAndServe(frontend, nil); e != nil {
				log.Fatalln("[ERROR]", "failed to listen HTTP:", e)
			}
		}()
	}

	closer := make(chan map[*net.TCPAddr]func(), 1)
	closer <- map[*net.TCPAddr]func(){}
	for i := range dsts {
		go func(i int) {
			for {
				if cl := <-closer; cl == nil {
					closer <- cl
					break
				} else {
					closer <- cl
				}

				binds[i] = &smpp.Bind{BindInfo: info}
				log.Println("[INFO]", "bind", i, ": connecting SMPP", binds[i].BindType, "bind to", dsts[i])
				c, e := net.DialTCP("tcp", localAddr, dsts[i])
				if e != nil {
					log.Println("[ERROR]", "bind", i, ": closed, error=", e)
					if o, ok := e.(*net.OpError); !ok || !o.Timeout() {
						time.Sleep(time.Second * 30)
					}
					continue
				}

				if cl := <-closer; cl != nil {
					cl[dsts[i]] = binds[i].Close
					closer <- cl
				} else {
					closer <- cl
					break
				}

				log.Println("[INFO]", "bind", i, ": closed, error=", binds[i].DialAndServe(c))
				binds[i] = nil

				if cl := <-closer; cl != nil {
					delete(cl, dsts[i])
					closer <- cl
				} else {
					closer <- cl
					break
				}
				time.Sleep(time.Second * 30)
			}
		}(i)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	if call := <-sigc; call != nil {
		log.Println("[INFO]", "closing binds")
		cl := <-closer
		for _, v := range cl {
			v()
		}
		closer <- nil

		for w := true; w; {
			w = false
			for _, b := range binds {
				if b != nil {
					w = true
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	}
}

func handleHTTP(w http.ResponseWriter, r *http.Request, req smpp.PDU, b []*smpp.Bind) {
	offset := rand.Intn(len(b))
	if b[offset] != nil && b[offset].IsActive() {
		dictionary.HandleHTTP(w, r, req, b[offset])
		return
	}

	for i := offset + 1; i < len(b); i++ {
		if b[i] != nil && b[i].IsActive() {
			dictionary.HandleHTTP(w, r, req, b[i])
			return
		}
	}
	for i := 0; i < offset; i++ {
		if b[i] != nil && b[i].IsActive() {
			dictionary.HandleHTTP(w, r, req, b[i])
			return
		}
	}

	data, _ := json.Marshal(struct {
		T string `json:"title"`
		D string `json:"detail"`
	}{T: "failed to send request", D: "no SMPP connection is available"})

	w.Header().Add("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(data)
}
