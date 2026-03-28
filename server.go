package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Parse JSON body into args
		args := map[string]interface{}{}
		if r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
				http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Inject request headers as __ow_headers with lowercase keys (OpenWhisk convention)
		owHeaders := make(map[string]interface{}, len(r.Header))
		for k, v := range r.Header {
			owHeaders[strings.ToLower(k)] = v[0]
		}
		args["__ow_headers"] = owHeaders

		result := Main(args)

		statusCode := http.StatusOK
		if code, ok := result["statusCode"].(int); ok {
			statusCode = code
		} else if code, ok := result["statusCode"].(float64); ok {
			statusCode = int(code)
		}
		delete(result, "statusCode")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	})

	log.Println("listening on :80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal(err)
	}
}
