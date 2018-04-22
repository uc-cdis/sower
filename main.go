package main

import (
    "fmt"
    "log"
    "net/http"
    "handlers"
)

func main() {
    fmt.Println("Running main")
    http.HandleFunc("/", repoHandler)
    http.HandleFunc("/_status", handlers.Status)
    http.HandleFunc("/_version", handlers.Version)
    log.Fatal(http.ListenAndServe("0.0.0.0:8000", nil))
}

func repoHandler(w http.ResponseWriter, r *http.Request) {
    //...
}
