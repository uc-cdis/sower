package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func RegisterSower() {
	http.HandleFunc("/dispatch", dispatch)
	http.HandleFunc("/status", status)
	http.HandleFunc("/list", list)
}

func dispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Not Found", 404)
		return
	}
	accessToken := getBearerToken(r)

	inputData, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	result, err := createK8sJob(string(inputData), *accessToken)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, string(out))
}

func status(w http.ResponseWriter, r *http.Request) {
	UID := r.URL.Query().Get("UID")
	if UID != "" {
		result, errUID := getJobStatusByID(UID)
		if errUID != nil {
			http.Error(w, errUID.Error(), 500)
			return
		}

		out, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		fmt.Fprintf(w, string(out))
	} else {
		http.Error(w, "Missing UID argument", 300)
		return
	}
}

func list(w http.ResponseWriter, r *http.Request) {
	result := listJobs(getJobClient())

	out, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, string(out))
}

func getBearerToken(r *http.Request) *string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil
	}
	s := strings.SplitN(authHeader, " ", 2)
	if len(s) == 2 && strings.ToLower(s[0]) == "bearer" {
		return &s[1]
	}
	return nil
}
