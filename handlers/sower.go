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

type Principal struct {
    Raw       string // original, e.g., "auth0|67ec..."
    LabelSafe string // sanitized for k8s label value
}

func dispatch(w http.ResponseWriter, r *http.Request) {
	log.Debug("Dispatch")
	if r.Method != "POST" {
		http.Error(w, "Not Found", 404)
		return
	}

	accessToken := getBearerToken(r)

	inputDataStr, err := io.ReadAll(r.Body)
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

	principal, err := getPrincipalFromToken(accessTokenVal)
	if err != nil {
		panic(err)
	}

	// pass both raw and safe; createK8sJob will still defensively sanitize at label write
	result, err := createK8sJob(currentAction, string(out), accessFormat, accessTokenVal, userName, principal.LabelSafe)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out, err = json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprint(w, string(out))
}

func status(w http.ResponseWriter, r *http.Request) {
    UID := r.URL.Query().Get("UID")
    if UID != "" {
        username := ""
        if bt := getBearerToken(r); bt != nil && *bt != "" {
            if p, err := getPrincipalFromToken(*bt); err == nil {
                username = p.LabelSafe
            }
        }

        result, errUID := getJobStatusByID(UID, sanitizeLabelValue(username))
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
        http.Error(w, "Missing UID argument", 400)
        return
    }
}



func output(w http.ResponseWriter, r *http.Request) {
	accessToken := getBearerToken(r)

	accessTokenVal := ""
	if accessToken != nil {
		accessTokenVal = *accessToken
	}

	principal, err := getPrincipalFromToken(accessTokenVal)
	if err != nil {
		panic(err.Error())
	}

	UID := r.URL.Query().Get("UID")
	if UID != "" {
		result, errUID := getJobLogs(UID, principal.LabelSafe)
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

	principal, err := getPrincipalFromToken(accessTokenVal)
	if err != nil {
		panic(err.Error())
	}

	result := listJobs(getJobClient(), principal.LabelSafe)

	out, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprint(w, string(out))
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

func getPrincipalFromToken(accessTokenVal string) (Principal, error) {
	jwksURL := "http://fence-service/.well-known/jwks"

	// create the JWKS from the resource at the given URL
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		log.Debugf("Failed to create JWKS from resource at the given URL.\nError: %s", err.Error())
		return Principal{}, err
	}

	token, err := jwt.Parse(accessTokenVal, jwks.Keyfunc)
	if err != nil {
		log.Debugf("Failed to parse the JWT.\nError: %s", err.Error())
		return Principal{}, err
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
            // This field was previously called "email" or "username"; it’s really a principal ID.
            raw := user["name"].(string) // often email-ish or provider-prefixed ID
			raw = strings.ReplaceAll(raw, "@", "_")
            return Principal{Raw: raw, LabelSafe: sanitizeLabelValue(raw)}, nil
		} else if claims["azp"] != nil {
			// Client token
            raw := claims["azp"].(string)
            return Principal{Raw: raw, LabelSafe: sanitizeLabelValue(raw)}, nil
		}
	}
	return Principal{}, nil
}
