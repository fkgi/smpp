package smpp

import "net"

var (
	// BoundNotify is called when SMPP bind is up.
	// Inputs are SMPP bind informations and peer address.
	BoundNotify func(BindInfo, net.Addr) = nil

	// UnboundNotify is called when SMPP bind is up.
	// Inputs are SMPP bind informations and peer address.
	UnboundNotify func(BindInfo, net.Addr) = nil

	// TraceTxMessage is called when Diameter message is sent.
	// Inputs are command ID, status code of response, sequence number,
	// message data and occured error while message handling.
	TraceTxMessage func(CommandID, StatusCode, uint32, []byte, error) = nil

	// TraceRxMessage is called when Diameter message is receved.
	// Inputs are command ID, status code of response, sequence number,
	// message data and occured error while message handling.
	TraceRxMessage func(CommandID, StatusCode, uint32, []byte, error) = nil
)
