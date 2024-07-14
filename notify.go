package smpp

var ConnectionDownNotify func(b *Bind) = nil
var RxMessageNotify func(id CommandID, stat, seq uint32, body []byte) = nil
var TxMessageNotify func(id CommandID, stat, seq uint32, body []byte) = nil
