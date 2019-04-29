package main

import (
    "fmt"
    "log"
    "net/http"
    "handlers"

    "github.com/gorilla/mux"
)

func main() {
    fmt.Println("Running main")
    /*
    http.HandleFunc("/", repoHandler)
    handlers.RegisterSystem()
    handlers.RegisterSower()
    */
    router := buildRouter()
    log.Fatal(http.ListenAndServe("0.0.0.0:8000", router))
}

func buildRouter() *mux.Router {
  r := mux.NewRouter()
  api := r.PathPrefix("/ga4gh/wes/v1").Subrouter()
  api.HandleFunc("/service-info", handlers.ServiceInfo) // in system.go
  api.HandleFunc("/runs", handlers.Runs) // in sower.go
  // api.HandleFunc("/runs/{id:[a-zA-Z0-9-]+}", runsIDHandler) // doesn't exist yet
  // api.HandleFunc("/runs/{id:[a-zA-Z0-9-]+}/cancel", cancelRunHandler) // doesn't exit yet
  api.HandleFunc("/runs/{id:[a-zA-Z0-9-]+}/status", handlers.Status) // in sower.go
  return r
}
