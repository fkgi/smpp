package smpp_test

import (
	"testing"

	"github.com/fkgi/smpp"
)

func TestBind(t *testing.T) {
	smpp.ID = "SVR"
	e := smpp.ListenAndServe(":2775")
	if e != nil {
		t.Fatal(e)
	}

	b, e := smpp.DialTransmitter("localhost:2775", smpp.BindInfo{
		SystemID:      "CRI",
		Password:      "passwod",
		SystemType:    "TEST",
		TypeOfNumber:  0x00,
		NumberingPlan: 0x00,
		AddressRange:  ""})
	if e != nil {
		t.Fatal(e)
	}

	b.Enquire()
	b.Submit()
	b.Close()
}
