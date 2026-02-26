package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/apex/log"
	"github.com/golang-jwt/jwt/v4"
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

	inputDataStr, err := io.ReadAll(r.Body)
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.WithError(err).Error("failed to close request body")
		}
	}()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	userName := r.Header.Get("REMOTE_USER")

	var inputRequest InputRequest
	_ = json.Unmarshal(inputDataStr, &inputRequest)

	var currentAction = inputRequest.Action

	var accessFormat = "presigned_url"

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

	email, err := getEmailFromToken(accessTokenVal)
	if err != nil {
		panic(err)
	}

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

	if _, err := fmt.Fprint(w, string(out)); err != nil {
		log.WithError(err).Error("failed to write response")
	}
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

		if _, err := fmt.Fprint(w, string(out)); err != nil {
			log.WithError(err).Error("failed to write response")
		}

	} else {
		http.Error(w, "Missing UID argument", http.StatusMultipleChoices)
		return
	}
}

func output(w http.ResponseWriter, r *http.Request) {
	accessToken := getBearerToken(r)

	accessTokenVal := ""
	if accessToken != nil {
		accessTokenVal = *accessToken
	}

	email, err := getEmailFromToken(accessTokenVal)
	if err != nil {
		panic(err.Error())
	}

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
				resLine = strings.ReplaceAll(logLine, "[out] ", "")
			}
		}

		jsonResLine := JobOutput{}
		jsonResLine.Output = resLine

		res, err := jsonResLine.JSON()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if _, err := fmt.Fprint(w, string(res)); err != nil {
			log.WithError(err).Error("failed to write response")
		}

	} else {
		http.Error(w, "Missing UID argument", http.StatusMultipleChoices)
		return
	}
}

func list(w http.ResponseWriter, r *http.Request) {
	accessToken := getBearerToken(r)

	accessTokenVal := ""
	if accessToken != nil {
		accessTokenVal = *accessToken
	}

	email, err := getEmailFromToken(accessTokenVal)
	if err != nil {
		panic(err.Error())
	}

	result := listJobs(getJobClient(), email)

	out, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if _, err := fmt.Fprint(w, string(out)); err != nil {
		log.WithError(err).Error("failed to write response")
	}
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

func getEmailFromToken(accessTokenVal string) (string, error) {
	jwksURL := "http://fence-service/.well-known/jwks"

	// create the JWKS from the resource at the given URL
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		log.Debugf("Failed to create JWKS from resource at the given URL.\nError: %s", err.Error())
		return "", err
	}

	token, err := jwt.Parse(accessTokenVal, jwks.Keyfunc)
	if err != nil {
		log.Debugf("Failed to parse the JWT.\nError: %s", err.Error())
		return "", err
	}

	// Verify if sub field exists, to identify if it is a user token or a client token
	// If sub exists, it is a user token, so we extract the email from the context field
	// Else-If azp exists, it is a client token, so we extract the client id from the azp field
	// If neither exists, return empty string

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["sub"] != nil {
			// User token
			context := claims["context"].(map[string]interface{})
			user := context["user"].(map[string]interface{})
			username := user["name"].(string)
			username = strings.ReplaceAll(username, "@", "_")
			return username, nil
		} else if claims["azp"] != nil {
			// Client token
			return claims["azp"].(string), nil
		}
	}
	return "", nil
}
