package smpp

import (
	"bytes"
)

type SubmitSM struct {
	smPDU
}

func (*SubmitSM) CommandID() CommandID     { return SubmitSm }
func (d *SubmitSM) MakeResponse() Response { return &SubmitSM_resp{} }

type SubmitSM_resp struct {
	MessageID string `json:"id"`
}

func (d *SubmitSM_resp) String() string     { return "| id: " + d.MessageID }
func (*SubmitSM_resp) CommandID() CommandID { return SubmitSmResp }

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
