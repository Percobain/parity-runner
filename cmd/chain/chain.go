package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/theblitlabs/parity-protocol/internal/config"
	"github.com/theblitlabs/parity-protocol/pkg/device"
	"github.com/theblitlabs/parity-protocol/pkg/logger"
)

func Run() {
	log := logger.Get()

	// Load config
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Get or generate device ID
	deviceID, err := device.VerifyDeviceID()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to verify device ID")
	}

	// Proxy request to the server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()

		// Strip /api prefix if present in the request
		path := strings.TrimPrefix(r.URL.Path, "/api")

		// Create new request to forward to the server
		targetURL := fmt.Sprintf("%s%s", cfg.Runner.ServerURL, path)
		log.Debug().
			Str("method", r.Method).
			Str("path", path).
			Str("target_url", targetURL).
			Msg("Forwarding request")

		// After reading the original body
		var requestData map[string]interface{}
		if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&requestData); err != nil {
			log.Error().Err(err).Msg("Failed to decode request body")
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Add device ID to request body
		requestData["creator_id"] = deviceID

		// After modifying the body
		modifiedBody, err := json.Marshal(requestData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal modified request body")
			http.Error(w, "Error processing request", http.StatusInternalServerError)
			return
		}

		// Create new request with modified body
		proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewBuffer(modifiedBody))
		if err != nil {
			http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
			return
		}

		// Copy headers again
		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(header, value)
			}
		}
		proxyReq.Header.Set("X-Device-ID", deviceID)
		proxyReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))

		// Forward the request
		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Error().Err(err).Msg("Error forwarding request")
			http.Error(w, "Error forwarding request", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for header, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}

		// Set response status code
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		io.Copy(w, resp.Body)
	})

	// Start local proxy server
	localAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, "3000")
	log.Info().
		Str("address", localAddr).
		Str("device_id", deviceID).
		Msg("Starting chain proxy server")

	if err := http.ListenAndServe(localAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("Failed to start chain proxy server")
	}
}
