package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/uc-cdis/sower/handlers/version"
)

type versionSummary struct {
	Commit  string `json:"commit"`
	Version string `json:"version"`
}

func RegisterSystem() {
	http.HandleFunc("/_status", systemStatus)
	http.HandleFunc("/_version", systemVersion)
}

func systemStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Healthy")
}

func systemVersion(w http.ResponseWriter, r *http.Request) {
	ver := versionSummary{Commit: version.GitCommit, Version: version.GitVersion}
	out, err := json.Marshal(ver)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprint(w, string(out))
}
