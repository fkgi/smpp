package main

import (
	"log"
	"net"
	"time"

	"github.com/fkgi/smpp"
)

func main() {
	smpp.ID = "SVR"
	l, e := net.Listen("tcp", ":2775")
	if e != nil {
		log.Fatalln(e)
	}
	go func() {
		_, e := smpp.Accept(l)
		if e != nil {
			log.Fatalln(e)
		}
	}()

	time.Sleep(time.Second)
	c, e := net.Dial("tcp", "localhost:2775")
	if e != nil {
		log.Fatalln(e)
	}
	b, e := smpp.Connect(c,
		smpp.BindInfo{
			BindType:      smpp.TRxBind,
			SystemID:      "CRI",
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

}
