package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fkgi/smpp"
	"github.com/fkgi/smpp/dictionary"
	"github.com/fkgi/teldata"
)

func main() {
	log.Println("[INFO]", "booting Round-Robin debugger for SMPP...")

	if dicData, e := dictionary.LoadDictionary(nil); e != nil {
		log.Fatalln("[ERROR]", "failed to read dictionary file:", e)
	} else {
		buf := new(strings.Builder)
		fmt.Fprint(buf, "supported parameter:")
		for _, p := range dicData.P {
			fmt.Fprintf(buf, " %s(%s/%s),", p.N, p.I, p.T)
		}
		log.Println("[INFO]", buf)
	}

	smpp.DefaultAlphabetIsGSM = os.Getenv("DEFAULT_ALPHABET") == "gsm7bit"
	if t, e := strconv.Atoi(os.Getenv("TIMEOUT")); e == nil {
		smpp.Expire = time.Duration(t) * time.Second
	}
	if os.Getenv("VERBOSE") != "yes" {
		smpp.TraceMessage = nil
	}

	frontend := os.Getenv("LOCALAPI_ADDR")
	if frontend == "" {
	} else if a, e := net.ResolveTCPAddr("tcp", frontend); e != nil {
		log.Fatalln("[ERROR]", "invalid HTTP local address:", e)
	} else {
		frontend = a.String()
		log.Println("[INFO]", "HTTP interface:", frontend)
	}

	if dictionary.Backend = os.Getenv("BACKENDAPI_ADDR"); dictionary.Backend == "" {
	} else if _, e := url.Parse("http://" + dictionary.Backend); e != nil {
		log.Fatalln("[ERROR]", "invalid HTTP backend host:", e)
	} else {
		dictionary.Backend = "http://" + dictionary.Backend
		log.Println("[INFO]", "HTTP backend:", dictionary.Backend)
		smpp.RequestHandler = dictionary.HandleSMPP
	}

	if smpp.ID = os.Getenv("SYSTEM_ID"); smpp.ID != "" {
	} else if h, e := os.Hostname(); e == nil {
		smpp.ID = h
	} else {
		smpp.ID = "roundrobin"
	}

	info := smpp.BindInfo{
		Password:   os.Getenv("PASSWORD"),
		SystemType: os.Getenv("TYPE")}
	if a := strings.SplitN(os.Getenv("ADDRESS"), ":", 3); len(a) != 3 {
		log.Fatalln("[ERROR]", "invalid ESME address format")
	} else if t, e := strconv.Atoi(a[0]); e != nil {
		log.Fatalln("[ERROR]", "invalid TON in ESME address:", e)
	} else if n, e := strconv.Atoi(a[1]); e != nil {
		log.Fatalln("[ERROR]", "invalid NPI in ESME address:", e)
	} else {
		info.AddressRange = a[2]
		info.TypeOfNumber = teldata.NatureOfAddress(t)
		info.NumberingPlan = teldata.NumberingPlan(n)
	}
	if len(frontend) != 0 && len(dictionary.Backend) != 0 {
		info.BindType = smpp.TRxBind
	} else if len(frontend) != 0 {
		info.BindType = smpp.TxBind
	} else if len(dictionary.Backend) != 0 {
		info.BindType = smpp.RxBind
	} else {
		log.Fatalln("[ERROR]", "no HTTP side information")
	}

	var localAddr *net.TCPAddr = nil
	if a := os.Getenv("LOCAL_ADDR"); a != "" {
		var e error
		if localAddr, e = net.ResolveTCPAddr("tcp", a+":0"); e != nil {
			log.Fatalln("[ERROR]", "invalid local address:", e)
		}
	}

	dsts := []*net.TCPAddr{}
	for i := range 10 {
		a := os.Getenv(fmt.Sprintf("PEER_ADDR%d", i))
		if a == "" {
			continue
		}
		if _, _, e := net.SplitHostPort(a); e != nil {
			a = a + ":2775"
		}
		if a, e := net.ResolveTCPAddr("tcp", a); e != nil {
			log.Fatalln("[ERROR]", "invalid destination address", e)
		} else {
			dsts = append(dsts, a)
		}
	}
	binds := make([]*smpp.Bind, len(dsts))

	if info.BindType != smpp.RxBind {
		http.HandleFunc("POST /smppmsg/v1/data",
			func(w http.ResponseWriter, r *http.Request) {
				handleHTTP(w, r, &smpp.DataSM{}, binds)
			})
		http.HandleFunc("POST /smppmsg/v1/submit",
			func(w http.ResponseWriter, r *http.Request) {
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
				var c net.Conn
				var e error
				if c, e = net.DialTCP("tcp", localAddr, dsts[i]); e != nil {
					binds[i] = nil
					log.Println("[ERROR]", "bind", i, ": closed, error=", e)
					if o, ok := e.(*net.OpError); !ok || !o.Timeout() {
						time.Sleep(time.Second * 30)
					}
					continue
				}
				if os.Getenv("TLS") == "yes" {
					c = tls.Client(c, &tls.Config{InsecureSkipVerify: true})
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
