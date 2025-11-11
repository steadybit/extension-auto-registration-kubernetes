package autoregistration

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

var MU = &sync.RWMutex{}
var AddedExtensions []string
var RemovedExtensions []string
var CurrentExtensions []string

func createMockAgent() *httptest.Server {
	MU.Lock()
	AddedExtensions = []string{}
	RemovedExtensions = []string{}
	CurrentExtensions = []string{}
	MU.Unlock()
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	server := httptest.Server{
		Listener: listener,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().Str("path", r.URL.Path).Str("method", r.Method).Str("query", r.URL.RawQuery).Msg("Request received")
			if strings.HasPrefix(r.URL.Path, "/extensions") && r.Method == http.MethodGet {
				var buf strings.Builder
				buf.WriteString("[")
				MU.RLock()
				for i, ext := range CurrentExtensions {
					buf.WriteString(ext)
					if i < len(CurrentExtensions)-1 {
						buf.WriteString(",")
					}
				}
				MU.RUnlock()
				buf.WriteString("]")
				response := buf.String()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(response))
			} else if strings.HasPrefix(r.URL.Path, "/extensions") && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				body, _ := io.ReadAll(r.Body)
				MU.Lock()
				AddedExtensions = append(AddedExtensions, string(body))
				CurrentExtensions = append(CurrentExtensions, string(body))
				MU.Unlock()
			} else if strings.HasPrefix(r.URL.Path, "/extensions") && r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
				body, _ := io.ReadAll(r.Body)
				MU.Lock()
				RemovedExtensions = append(RemovedExtensions, string(body))
				for i, ext := range CurrentExtensions {
					if ext == string(body) {
						CurrentExtensions = append(CurrentExtensions[:i], CurrentExtensions[i+1:]...)
						break
					}
				}
				MU.Unlock()
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		})},
	}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Agent")
	return &server
}
