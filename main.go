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
    handlers.RegisterSystem()
    handlers.RegisterSower()
    log.Fatal(http.ListenAndServe("0.0.0.0:8000", nil))
}

func repoHandler(w http.ResponseWriter, r *http.Request) {
    //...
}
