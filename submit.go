package smpp

import (
	"bytes"
	"fmt"
	"strings"
)

type SubmitSM struct {
	smPDU
}

func (d *SubmitSM) String() string {
	buf := new(strings.Builder)
	d.smPDU.WriteTo(buf)
	return buf.String()
}

func (*SubmitSM) CommandID() CommandID {
	return SubmitSm
}

func (d *SubmitSM) Marshal() []byte {
	return d.smPDU.Marshal()
}

func (d *SubmitSM) Unmarshal(data []byte) (e error) {
	return d.smPDU.Unmarshal(data)
}

func (d *SubmitSM) MakeResponse() Response {
	return &SubmitSM_resp{}
}

type SubmitSM_resp struct {
	MessageID string `json:"id"`
}

func (d *SubmitSM_resp) String() string {
	buf := new(strings.Builder)
	fmt.Fprint(buf, "| id:", d.MessageID)
	return buf.String()
}

func (*SubmitSM_resp) CommandID() CommandID {
	return SubmitSmResp
}

func (d *SubmitSM_resp) Marshal() []byte {
	w := bytes.Buffer{}
	if len(d.MessageID) != 0 {
		writeCString([]byte(d.MessageID), &w)
	}
	return w.Bytes()
}

func (d *SubmitSM_resp) Unmarshal(data []byte) (e error) {
	buf := bytes.NewBuffer(data)
	d.MessageID, e = readCString(buf)
	return
}
