package handlers

import (
    "fmt"
    "net/http"
)

func RegisterSower() {
    http.HandleFunc("/dispatch", dispatch)
    http.HandleFunc("/status", status)
}

func dispatch(w http.ResponseWriter, r *http.Request) {
    /*if (r.Method != "POST") {
       http.Error(w, "Not Found", 404)
       return
    }*/
    createK8sJob()
    fmt.Fprintf(w, "Dispatched")
}

func status(w http.ResponseWriter, r *http.Request) {
   fmt.Fprintf(w, "Running")
}
