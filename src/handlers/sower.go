package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Creds struct {
	BucketName string `json:"manifest_bucket_name"`
	Hostname   string `json:"hostname"`
	Key        string `json:"aws_access_key_id"`
	Secret     string `json:"aws_secret_access_key"`
}

func RegisterSower() {
	http.HandleFunc("/dispatch", dispatch)
	http.HandleFunc("/status", status)
	http.HandleFunc("/list", list)
	http.HandleFunc("/output", output)
}

func dispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Not Found", 404)
		return
	}

	credsFile, _ := ioutil.ReadFile("/creds.json")
	var creds Creds
	json.Unmarshal(credsFile, &creds)

	fmt.Println("successfully read creds.json")
	fmt.Println("s3 key:    " + creds.Key)
	fmt.Println("s3 secret: " + creds.Secret)

	accessToken := getBearerToken(r)

	inputData, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	userName := r.Header.Get("REMOTE_USER")

	result, err := createK8sJob(string(inputData), *accessToken, creds.Key, creds.Secret, userName)
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

func output(w http.ResponseWriter, r *http.Request) {
	UID := r.URL.Query().Get("UID")
	if UID != "" {
		result, errUID := getJobLogs(UID)
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
	fmt.Println("header: ", authHeader)
	if authHeader == "" {
		return nil
	}
	s := strings.SplitN(authHeader, " ", 2)
	if len(s) == 2 && strings.ToLower(s[0]) == "bearer" {
		return &s[1]
	}
	return nil
}
