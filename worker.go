package smpp

import (
	"fmt"
	"time"
)

const (
	minWorkers = 128
	maxWorkers = 65535 - minWorkers
)

var sharedQ = make(chan message, maxWorkers)
var activeWorkers = make(chan int, 1)

func init() {
	activeWorkers <- 0
	worker := func() {
		for c := 0; c < 500; {
			if len(sharedQ) < minWorkers {
				time.Sleep(time.Millisecond * 10)
				c++
				continue
			}
			if req, ok := <-sharedQ; !ok {
				break
			} else {
				handleMsg(req)
				c = 0
			}
		}
		activeWorkers <- (<-activeWorkers - 1)
	}
	for i := 0; i < minWorkers; i++ {
		go func() {
			for req, ok := <-sharedQ; ok; req, ok = <-sharedQ {
				a := <-activeWorkers
				activeWorkers <- a
				if len(sharedQ) > minWorkers && a < maxWorkers {
					activeWorkers <- (<-activeWorkers + 1)
					go worker()
				}
				handleMsg(req)
			}
		}()
	}
}

func handleMsg(msg message) {
	var req Request
	var res Response

	switch msg.id {
	case SubmitSm:
		req = &SubmitSM{}
		res = &SubmitSM_resp{}
	case DeliverSm:
		req = &DeliverSM{}
		res = &DeliverSM_resp{}
	case DataSm:
		req = &DataSM{}
		res = &DataSM_resp{}
	}
	if req == nil {
		panic(fmt.Sprintf("unexpected request PDU (ID:%#x)", msg.id))
	}

	stat := StatSysErr
	if e := req.Unmarshal(msg.body); e != nil || RequestHandler == nil {
	} else if stat, res = RequestHandler(msg.bind.BindInfo, req); res == nil {
		msg.bind.eventQ <- message{
			id:       GenericNack,
			stat:     stat,
			seq:      msg.seq,
			callback: dummyCallback}
	} else {
		msg.bind.eventQ <- message{
			id:       res.CommandID(),
			stat:     stat,
			seq:      msg.seq,
			body:     res.Marshal(),
			callback: dummyCallback}
	}
}

var dummyCallback = make(chan message)

var RequestHandler = func(info BindInfo, pdu Request) (StatusCode, Response) {
	/*
		const l = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

		id := make([]byte, 16)
		for i := range id {
			id[i] = l[rand.Intn(len(l))]
		}
		switch pdu.(type) {
		case *DataSM:
			return &DataSM_resp{
				MessageID: string(id),
			}
		case *SubmitSM:
			return &SubmitSM_resp{
				MessageID: string(id),
			}
		case *DeliverSM:
			return &DeliverSM_resp{}
		}
	*/
	return StatSysErr, nil
}
