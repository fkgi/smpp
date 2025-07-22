package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/fkgi/smpp"
)

func init() {
	smpp.TraceMessage = func(di smpp.Direction, id smpp.CommandID, stat smpp.StatusCode, seq uint32, body []byte) {
		var req smpp.Request
		var res smpp.Response
		switch id {
		case smpp.SubmitSm:
			req = &smpp.SubmitSM{}
		case smpp.SubmitSmResp:
			res = &smpp.SubmitSM_resp{}
		case smpp.DeliverSm:
			req = &smpp.DeliverSM{}
		case smpp.DeliverSmResp:
			res = &smpp.DeliverSM_resp{}
		case smpp.DataSm:
			req = &smpp.DataSM{}
		case smpp.DataSmResp:
			res = &smpp.DataSM_resp{}
		}

		if req != nil {
			if e := req.Unmarshal(body); e != nil {
				log.Printf("[INFO] %s SMPP message(sequence=%d) %s\n| %s", di, seq, id, e)
			} else {
				log.Printf("[INFO] %s SMPP message(sequence=%d) %s", di, seq, req)
			}
		} else if res != nil {
			if e := res.Unmarshal(body); e != nil {
				log.Printf("[INFO] %s SMPP message(sequence=%d) %s\n| %s", di, seq, id, e)
			} else {
				log.Printf("[INFO] %s SMPP message(sequence=%d) %s", di, seq, res)
			}
		} else {
			panic(fmt.Sprintf("unexpected request PDU (ID:%#x)", id))
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
