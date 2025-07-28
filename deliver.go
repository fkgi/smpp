package smpp

import (
	"bytes"
	"strings"
)

type DeliverSM struct {
	smPDU
}

func (d *DeliverSM) String() string {
	buf := new(strings.Builder)
	d.smPDU.WriteTo(buf)
	return buf.String()
}

func (*DeliverSM) CommandID() CommandID {
	return DeliverSm
}

func (d *DeliverSM) Marshal() []byte {
	return d.smPDU.Marshal()
}

func (d *DeliverSM) Unmarshal(data []byte) (e error) {
	return d.smPDU.Unmarshal(data)
}

func (d *DeliverSM) MakeResponse() Response {
	return &DeliverSM_resp{}
}

type DeliverSM_resp struct {
}

func (d *DeliverSM_resp) String() string {
	buf := new(strings.Builder)
	// fmt.Fprint(buf, "| id:", d.MessageID)
	return buf.String()
}

func (*DeliverSM_resp) CommandID() CommandID {
	return DeliverSmResp
}

func (d *DeliverSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	writeCString([]byte{}, &w)
	return w.Bytes()
}

func (d *DeliverSM_resp) Unmarshal(data []byte) (e error) {
	return
}
