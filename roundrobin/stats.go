package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/fkgi/smpp"
)

var rxMetrics = make(chan map[string]map[string]uint64, 1)
var txMetrics = make(chan map[string]map[string]uint64, 1)

func init() {
	rxMetrics <- map[string]map[string]uint64{}
	txMetrics <- map[string]map[string]uint64{}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	sysmetricsHandler(w)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Fprintln(w, "# HELP heap_alloc_bytes Number of heap bytes allocated and still in use")
	fmt.Fprintln(w, "# TYPE heap_alloc_bytes gauge")
	fmt.Fprintf(w, "heap_alloc_bytes %d\n", ms.Alloc)

	fmt.Fprintln(w, "# HELP roundrobin_active_worker_count Count of active worker")
	fmt.Fprintln(w, "# TYPE roundrobin_active_worker gauge")
	fmt.Fprintf(w, "roundrobin_active_worker %d\n", smpp.ActiveWorkerCount())

	if len(binds) != 0 {
		fmt.Fprintln(w, "# HELP roundrobin_bind_state Status of ASP(down=0, active=1)")
		fmt.Fprintln(w, "# TYPE roundrobin_bind_state gauge")
		for i, b := range binds {
			if b == nil || !b.IsActive() {
				fmt.Fprintf(w, "roundrobin_bind_state{address=\"%s\"} 0\n", dsts[i].String())
			} else {
				fmt.Fprintf(w, "roundrobin_bind_state{address=\"%s\"} 1\n", dsts[i].String())
			}
		}
	}

	m := <-txMetrics
	if len(m) != 0 {
		fmt.Fprintln(w, "# HELP roundrobin_tx_pdu Total number of outgoing PDU")
		fmt.Fprintln(w, "# TYPE roundrobin_tx_pdu counter")
		for k, v := range m {
			for c, v := range v {
				fmt.Fprintf(w, "roundrobin_tx_pdu{message=\"%s\",code=\"%s\"} %d\n", k, c, v)
			}
		}
	}
	txMetrics <- m
	m = <-rxMetrics
	if len(m) != 0 {
		fmt.Fprintln(w, "# HELP roundrobin_rx_pdu Total number of outgoing PDU")
		fmt.Fprintln(w, "# TYPE roundrobin_rx_pdu counter")
		for k, v := range m {
			for c, v := range v {
				fmt.Fprintf(w, "roundrobin_rx_pdu{message=\"%s\",code=\"%s\"} %d\n", k, c, v)
			}
		}
	}
	rxMetrics <- m
}

func count(id smpp.CommandID, st smpp.StatusCode, rx bool) {
	var mx map[string]map[string]uint64
	if rx {
		mx = <-rxMetrics
	} else {
		mx = <-txMetrics
	}

	m := mx[id.String()]
	if m == nil {
		m = map[string]uint64{}
	}
	m[st.String()] = m[st.String()] + 1
	mx[id.String()] = m

	if rx {
		rxMetrics <- mx
	} else {
		txMetrics <- mx
	}
}
