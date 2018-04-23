package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
)

func RegisterSower() {
    http.HandleFunc("/dispatch", dispatch)
    http.HandleFunc("/status", status)
    http.HandleFunc("/list", list)
}

func dispatch(w http.ResponseWriter, r *http.Request) {
    /*if (r.Method != "POST") {
       http.Error(w, "Not Found", 404)
       return
    }*/
    result, err := createK8sJob()
    if err != nil {
        http.Error(w, result, 500)
        return
    }
    fmt.Fprintf(w, result)
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
