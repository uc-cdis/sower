package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apex/log"
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
	if _, err := fmt.Fprint(w, "Healthy"); err != nil {
		log.WithError(err).Error("failed to write systemStatus response")
	}
}

func systemVersion(w http.ResponseWriter, r *http.Request) {
	ver := versionSummary{Commit: version.GitCommit, Version: version.GitVersion}
	out, err := json.Marshal(ver)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if _, err := fmt.Fprint(w, string(out)); err != nil {
		log.WithError(err).Error("failed to write systemVersion response")
	}
}
