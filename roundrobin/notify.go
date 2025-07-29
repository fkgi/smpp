package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/fkgi/smpp"
)

func init() {
	smpp.TraceMessage = func(di smpp.Direction, id smpp.CommandID, stat smpp.StatusCode, seq uint32, body []byte) {
		if id.IsRequest() {
			var req smpp.Request
			switch id {
			case smpp.SubmitSm:
				req = &smpp.SubmitSM{}
			case smpp.DeliverSm:
				req = &smpp.DeliverSM{}
			case smpp.DataSm:
				req = &smpp.DataSM{}
			default:
				log.Printf("[INFO] %s %s (seq=%d)", di, di, seq)
				return
			}
			if e := req.Unmarshal(body); e != nil {
				log.Printf("[INFO] %s %s (seq=%d)\n| %s", di, id, seq, e)
			} else {
				log.Printf("[INFO] %s %s (seq=%d)\n%s", di, id, seq, req)
			}
		} else {
			var res smpp.Response
			switch id {
			case smpp.SubmitSmResp:
				res = &smpp.SubmitSM_resp{}
			case smpp.DeliverSmResp:
				res = &smpp.DeliverSM_resp{}
			case smpp.DataSmResp:
				res = &smpp.DataSM_resp{}
			default:
				log.Printf("[INFO] %s %s (seq=%d, stat=%s)", di, id, seq, stat)
				return
			}
			if e := res.Unmarshal(body); e != nil {
				log.Printf("[INFO] %s %s (seq=%d, stat=%s)\n| %s", di, id, seq, stat, e)
			} else {
				log.Printf("[INFO] %s %s (seq=%d, stat=%s)\n%s", di, id, seq, stat, res)
			}
		}
	}

	smpp.BoundNotify = func(i smpp.BindInfo) {
		buf := new(strings.Builder)
		fmt.Fprintln(buf, "bind is up")
		fmt.Fprintln(buf, "| peer system ID  :", i.PeerID)
		fmt.Fprint(buf, "| bind type       : ", i.BindType)
		if *bindType == "svr" {
			fmt.Fprintln(buf)
			fmt.Fprintln(buf, "| ESME password   :", i.Password)
			fmt.Fprintln(buf, "| ESME system type:", i.SystemType)
			fmt.Fprintf(buf, "| ESME address    : %s(ton=%d, npi=%d)",
				i.AddressRange, i.TypeOfNumber, i.NumberingPlan)
		}
		log.Println("[INFO]", buf)
	}
}
