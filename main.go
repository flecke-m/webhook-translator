package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func Main(args map[string]interface{}) map[string]interface{} {
	// In DigitalOcean Functions (OpenWhisk) web actions, HTTP request headers
	// are available under the __ow_headers key as a lowercase-keyed map.
	headers := map[string]interface{}{}
	if h, ok := args["__ow_headers"].(map[string]interface{}); ok {
		headers = h
	}

	getHeader := func(key string) string {
		v, _ := headers[strings.ToLower(key)].(string)
		return v
	}

	ntfyURL := getHeader("ntfy_url")
	if ntfyURL == "" {
		ntfyURL = "https://ntfy.sh"
	}

	topic := getHeader("topic")
	if topic == "" {
		log.Println("[webhook-translator] ERROR: missing required header: topic")
		return map[string]interface{}{
			"statusCode": 400,
			"body":       "missing required header: topic",
		}
	}

	log.Printf("[webhook-translator] incoming call: topic=%s ntfy_url=%s", topic, ntfyURL)

	// Image is in alarm.thumbnail as a data URL (data:image/jpeg;base64,...)
	imageBase64 := ""
	if alarm, ok := args["alarm"].(map[string]interface{}); ok {
		imageBase64, _ = alarm["thumbnail"].(string)
	}
	var req *http.Request
	var err error

	if imageBase64 == "" {
		// No picture available — send a plain-text message instead
		log.Printf("[webhook-translator] no picture found, forwarding text message to %s/%s", strings.TrimRight(ntfyURL, "/"), topic)
		req, err = http.NewRequest(http.MethodPost, strings.TrimRight(ntfyURL, "/")+"/"+topic, strings.NewReader("No picture"))
		if err != nil {
			return map[string]interface{}{
				"statusCode": 500,
				"body":       "failed to create request: " + err.Error(),
			}
		}
		req.Header.Set("Content-Type", "text/plain")
	} else {
		// Derive filename from the data URL mime type, then strip the prefix
		filename := "image.jpg"
		if strings.HasPrefix(imageBase64, "data:") {
			if semi := strings.Index(imageBase64, ";"); semi >= 0 {
				filename = mimeTypeToFilename(imageBase64[5:semi])
			}
			if comma := strings.Index(imageBase64, ","); comma >= 0 {
				imageBase64 = imageBase64[comma+1:]
			}
		}

		data, decErr := base64.StdEncoding.DecodeString(imageBase64)
		if decErr != nil {
			return map[string]interface{}{
				"statusCode": 400,
				"body":       "invalid base64 payload: " + decErr.Error(),
			}
		}

		log.Printf("[webhook-translator] picture found (%s, %d bytes), forwarding to %s/%s", filename, len(data), strings.TrimRight(ntfyURL, "/"), topic)
		req, err = http.NewRequest(http.MethodPut, strings.TrimRight(ntfyURL, "/")+"/"+topic, bytes.NewReader(data))
		if err != nil {
			return map[string]interface{}{
				"statusCode": 500,
				"body":       "failed to create request: " + err.Error(),
			}
		}

		req.Header.Set("Filename", filename)
		req.Header.Set("Content-Type", detectContentType(filename))
	}

	// Forward Title, Tags and Authorization headers from the incoming request to ntfy
	if title := getHeader("title"); title != "" {
		req.Header.Set("Title", title)
	}
	if tags := getHeader("tags"); tags != "" {
		req.Header.Set("Tags", tags)
	}
	if auth := getHeader("authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[webhook-translator] ERROR: publish to ntfy failed: %v", err)
		return map[string]interface{}{
			"statusCode": 502,
			"body":       "publish failed: " + err.Error(),
		}
	}
	defer resp.Body.Close()
	log.Printf("[webhook-translator] ntfy responded with status %d", resp.StatusCode)

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return map[string]interface{}{
			"statusCode": resp.StatusCode,
			"body":       "ntfy returned non-JSON response",
		}
	}

	out["statusCode"] = resp.StatusCode
	return out
}

func mimeTypeToFilename(mimeType string) string {
	switch mimeType {
	case "image/png":
		return "image.png"
	case "image/gif":
		return "image.gif"
	case "image/webp":
		return "image.webp"
	case "image/bmp":
		return "image.bmp"
	case "image/svg+xml":
		return "image.svg"
	default:
		return "image.jpg"
	}
}

func detectContentType(filename string) string {
	name := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".gif"):
		return "image/gif"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
	case strings.HasSuffix(name, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	default:
		return "image/jpeg"
	}
}