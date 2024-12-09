package main

import (
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/uc-cdis/sower/handlers"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	// "github.com/apex/log/handlers/json"
)

func main() {
	log.SetHandler(text.New(os.Stderr))

	rand.New(rand.NewSource(time.Now().UnixNano()))

	log.Info("Server started")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	handlers.RegisterSystem()
	handlers.RegisterSower()
	go handlers.StartMonitoringProcess()

	log.Fatalf("%s", http.ListenAndServe("0.0.0.0:8000", nil))
}
