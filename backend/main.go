package main

import "net/http"

func main() {
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":0,"id":"dummy-id"}`))
	})
	http.HandleFunc("/deliver", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":0,"id":"dummy-id"}`))
	})
	http.ListenAndServe(":8082", nil)
}
