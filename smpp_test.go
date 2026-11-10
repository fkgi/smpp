package smpp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestPDUFrameRoundTrip(t *testing.T) {
	want := message{
		id:   SubmitSm,
		stat: StatSysErr,
		seq:  42,
		body: []byte{0x01, 0x02, 0x03},
	}

	var wire bytes.Buffer
	writer := bufio.NewReadWriter(bufio.NewReader(&wire), bufio.NewWriter(&wire))
	if err := writePDU(writer, want); err != nil {
		t.Fatalf("writePDU() error = %v", err)
	}

	reader := bufio.NewReadWriter(bufio.NewReader(&wire), bufio.NewWriter(io.Discard))
	got, err := readPDU(reader)
	if err != nil {
		t.Fatalf("readPDU() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readPDU() = %#v, want %#v", got, want)
	}
}

func TestReadPDURejectsInvalidAndTruncatedFrames(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
	}{
		{name: "short header", wire: []byte{0, 0, 0, 15}},
		{name: "truncated command id", wire: []byte{0, 0, 0, 16, 0}},
		{name: "truncated body", wire: []byte{
			0, 0, 0, 17,
			0, 0, 0, 4,
			0, 0, 0, 0,
			0, 0, 0, 1,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReadWriter(bufio.NewReader(bytes.NewReader(tt.wire)), bufio.NewWriter(io.Discard))
			if _, err := readPDU(reader); err == nil {
				t.Fatal("readPDU() error = nil, want an error")
			}
		})
	}
}

func TestSubmitSMRoundTrip(t *testing.T) {
	want := &SubmitSM{smPDU: smPDU{
		SvcType:              "sms",
		SrcTON:               1,
		SrcNPI:               1,
		SrcAddr:              "1234",
		DstTON:               1,
		DstNPI:               1,
		DstAddr:              "5678",
		ProtocolId:           2,
		PriorityFlag:         1,
		ScheduleDeliveryTime: "260101000000000+",
		ValidityPeriod:       "260102000000000+",
		RegisteredDelivery: registeredDelivery{
			Receipt:           ReceiptOnAll,
			DeliveryAck:       true,
			ManualUserAck:     true,
			IntermediateNotif: true,
		},
		ReplaceIfPresentFlag: true,
		DataCoding:           3,
		SmDefaultMsgId:       4,
		ShortMessage:         UserData{Text: "hello"},
		Param:                OptionalParameters{0x0204: {0x01, 0x02}},
	}}

	got := &SubmitSM{}
	if err := got.Unmarshal(want.Marshal(0x34)); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %#v, want %#v", got, want)
	}
}

func TestDataSMRoundTrip(t *testing.T) {
	want := &DataSM{
		SvcType:    "sms",
		SrcTON:     1,
		SrcNPI:     1,
		SrcAddr:    "1234",
		DstTON:     1,
		DstNPI:     1,
		DstAddr:    "5678",
		DataCoding: 3,
		Param: OptionalParameters{
			0x0204: {0x01, 0x02},
		},
	}

	got := &DataSM{}
	if err := got.Unmarshal(want.Marshal(0x34)); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %#v, want %#v", got, want)
	}
}

func TestBindRequestRoundTrip(t *testing.T) {
	want := &bindReq{
		cmd:        BindTransceiver,
		SystemID:   "client",
		Password:   "secret",
		SystemType: "sms",
		Version:    0x34,
		AddrTON:    1,
		AddrNPI:    1,
		AddrRange:  "123*",
	}
	got := &bindReq{cmd: want.cmd}
	if err := got.Unmarshal(want.Marshal(0x34)); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %#v, want %#v", got, want)
	}
}

func TestMakePDUof(t *testing.T) {
	tests := []CommandID{
		GenericNack,
		BindReceiver, BindReceiverResp,
		BindTransmitter, BindTransmitterResp,
		BindTransceiver, BindTransceiverResp,
		SubmitSm, SubmitSmResp,
		DeliverSm, DeliverSmResp,
		Unbind, UnbindResp,
		EnquireLink, EnquireLinkResp,
		DataSm, DataSmResp,
	}
	for _, id := range tests {
		t.Run(id.String(), func(t *testing.T) {
			pdu := MakePDUof(id)
			if pdu == nil {
				t.Fatal("MakePDUof() = nil")
			}
			if got := pdu.CommandID(); got != id {
				t.Errorf("CommandID() = %v, want %v", got, id)
			}
		})
	}
	if pdu := MakePDUof(QuerySm); pdu != nil {
		t.Errorf("MakePDUof(QuerySm) = %T, want nil", pdu)
	}
}

func TestOptionalParametersJSONRoundTrip(t *testing.T) {
	want := OptionalParameters{0x0204: {0x01, 0x02}}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got OptionalParameters
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %#v, want %#v", got, want)
	}
}

func TestOptionalParametersJSONRejectsInvalidParameter(t *testing.T) {
	tests := []string{
		`{"0x0204":"0102"}`,
		`{"0204":123}`,
		`{"0204":"xyz"}`,
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			var params OptionalParameters
			if err := json.Unmarshal([]byte(input), &params); err == nil {
				t.Fatal("Unmarshal() error = nil, want an error")
			}
		})
	}
}

func TestReadCString(t *testing.T) {
	got, err := readCString(bytes.NewBuffer([]byte("value\x00tail")))
	if err != nil {
		t.Fatalf("readCString() error = %v", err)
	}
	if got != "value" {
		t.Errorf("readCString() = %q, want %q", got, "value")
	}

	_, err = readCString(bytes.NewBufferString("unterminated"))
	if !errors.Is(err, io.EOF) {
		t.Errorf("readCString() error = %v, want %v", err, io.EOF)
	}
}
