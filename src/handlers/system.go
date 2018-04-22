package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type versionSummary struct {
    Commit       string
    Version      string
}

func Status(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Healthy")
}

func Version(w http.ResponseWriter, r *http.Request) {
    ver := versionSummary{Commit: gitcommit, Version: gitversion}
    out, err := json.Marshal(ver)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    fmt.Fprintf(w, string(out))
}
