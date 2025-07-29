package smpp

import (
	"bytes"
)

type DeliverSM struct {
	smPDU
}

func (*DeliverSM) CommandID() CommandID     { return DeliverSm }
func (d *DeliverSM) MakeResponse() Response { return &DeliverSM_resp{} }

type DeliverSM_resp struct{}

func (d *DeliverSM_resp) String() string     { return "" }
func (*DeliverSM_resp) CommandID() CommandID { return DeliverSmResp }

func (d *DeliverSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	writeCString([]byte{}, &w)
	return w.Bytes()
}

func (d *DeliverSM_resp) Unmarshal(data []byte) (e error) {
	return
}
