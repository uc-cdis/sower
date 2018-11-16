package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type postData struct {
	InputURL  string `json:"inputURL"`
	OutputURL string `json:"outputURL"`
}

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
	decoder := json.NewDecoder(r.Body)
	fmt.Println(r.Body)

	var data postData
	err := decoder.Decode(&data)
	if err != nil {
		http.Error(w, "Failed to decode JSON", 400)
		return
	}
	fmt.Println("input URL: ", data.InputURL)
	fmt.Println("output URL: ", data.OutputURL)
	result, err := createK8sJob(data.InputURL, data.OutputURL)
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
