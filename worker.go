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
	switch msg.id {
	case SubmitSm:
		req = &SubmitSM{}
	case DeliverSm:
		req = &DeliverSM{}
	case DataSm:
		req = &DataSM{}
	}
	if req == nil {
		panic(fmt.Sprintf("unexpected request PDU (ID:%#x)", msg.id))
	}

	stat := StatSysErr
	var res Response
	if e := req.Unmarshal(msg.body); e != nil || RequestHandler == nil {
		res = req.MakeResponse()
	} else if stat, res = RequestHandler(msg.bind.BindInfo, req); res == nil {
		res = req.MakeResponse()
	}
	msg.bind.eventQ <- message{
		id:       res.CommandID(),
		stat:     stat,
		seq:      msg.seq,
		body:     res.Marshal(),
		callback: dummyCallback}
}

var dummyCallback = make(chan message)
