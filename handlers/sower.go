package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/apex/log"
	"github.com/golang-jwt/jwt"
)

func RegisterSower() {
	http.HandleFunc("/dispatch", dispatch)
	http.HandleFunc("/status", status)
	http.HandleFunc("/list", list)
	http.HandleFunc("/output", output)
}

// InputRequest Struct
type InputRequest struct {
	Action string                 `json:"action"`
	Input  map[string]interface{} `json:"input"`
	Format string                 `json:"access_format"`
}

func dispatch(w http.ResponseWriter, r *http.Request) {
	log.Debug("Dispatch")
	if r.Method != "POST" {
		http.Error(w, "Not Found", 404)
		return
	}

	accessToken := getBearerToken(r)

	inputDataStr, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	userName := r.Header.Get("REMOTE_USER")

	var inputRequest InputRequest
	_ = json.Unmarshal(inputDataStr, &inputRequest)

	var currentAction = inputRequest.Action

	var accessFormat string = "presigned_url"

	if inputRequest.Format != "" {
		accessFormat = inputRequest.Format
	}

	out, err := json.Marshal(inputRequest.Input)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	accessTokenVal := ""
	if accessToken != nil {
		accessTokenVal = *accessToken
	}

	email := getEmailFromToken(accessTokenVal)

	result, err := createK8sJob(currentAction, string(out), accessFormat, accessTokenVal, userName, email)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out, err = json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, string(out))
}

func status(w http.ResponseWriter, r *http.Request) {
	email := ""

	UID := r.URL.Query().Get("UID")
	if UID != "" {
		result, errUID := getJobStatusByID(UID, email)
		if errUID != nil {
			http.Error(w, errUID.Error(), 500)
			return
		}

		out, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		fmt.Fprint(w, string(out))
	} else {
		http.Error(w, "Missing UID argument", 300)
		return
	}
}

func output(w http.ResponseWriter, r *http.Request) {
	accessToken := getBearerToken(r)

	accessTokenVal := ""
	if accessToken != nil {
		accessTokenVal = *accessToken
	}

	email := getEmailFromToken(accessTokenVal)

	UID := r.URL.Query().Get("UID")
	if UID != "" {
		result, errUID := getJobLogs(UID, email)
		if errUID != nil {
			http.Error(w, errUID.Error(), 500)
			return
		}

		_, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var resLine string
		newLineSep := func(c rune) bool {
			return c == '\n'
		}
		logLines := strings.FieldsFunc(result.Output, newLineSep)
		for _, logLine := range logLines {
			if strings.Contains(logLine, "[out] ") {
				resLine = strings.Replace(logLine, "[out] ", "", -1)
			}
		}

		jsonResLine := JobOutput{}
		jsonResLine.Output = resLine

		res, err := jsonResLine.JSON()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		fmt.Fprint(w, string(res))
	} else {
		http.Error(w, "Missing UID argument", 300)
		return
	}
}

func list(w http.ResponseWriter, r *http.Request) {
	accessToken := getBearerToken(r)

	accessTokenVal := ""
	if accessToken != nil {
		accessTokenVal = *accessToken
	}

	email := getEmailFromToken(accessTokenVal)

	result := listJobs(getJobClient(), email)

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

func getEmailFromToken(accessTokenVal string) string {
	var hmacSampleSecret []byte
	token, err := jwt.Parse(accessTokenVal, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return hmacSampleSecret, nil
	})

	email := "bad email"
	// the check should be
	// ok && token.Valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		fmt.Println(claims["context"])
		context := claims["context"].(map[string]interface{})
		user := context["user"].(map[string]interface{})
		email = user["name"].(string)
	} else {
		fmt.Println(err)
	}
	return strings.ReplaceAll(email, "@", "_")
}
