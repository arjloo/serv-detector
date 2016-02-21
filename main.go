package main

import (
	"time"

	"./discover"

	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	var m *discover.Monitor = nil
	for m == nil {
		m = discover.NewMonitor([]string{"http://192.168.0.2:4001"})

		if m == nil {
			time.Sleep(1)
		}
	}
	go m.WatchNodes()

	// start http server
	r := mux.NewRouter()
	r.HandleFunc("/v1.0/tenant", m.PostMonitorCfgHandler).Methods("POST")
	r.HandleFunc("/v1.0/service/{serv_name}", m.GetServInfoHandler).Methods("GET")
	http.Handle("/", r)
	http.ListenAndServe(":7171", nil)
}
