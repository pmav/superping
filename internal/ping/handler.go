package ping

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) http.Handler {
	handler := &Handler{store: store}
	return http.HandlerFunc(handler.logRequest)
}

func (handler *Handler) logRequest(response http.ResponseWriter, request *http.Request) {
	start := time.Now()
	loggedResponse := &loggingResponseWriter{ResponseWriter: response}
	handler.handle(loggedResponse, request)
	log.Printf("api method=%s path=%s status=%d duration=%s", request.Method, request.URL.Path, loggedResponse.status, time.Since(start).Round(time.Microsecond))
}

func (handler *Handler) handle(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/pings" {
		switch request.Method {
		case http.MethodGet:
			handler.list(response, request)
		case http.MethodPost:
			handler.add(response, request)
		default:
			response.Header().Set("Allow", "GET, POST")
			writeError(response, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if strings.HasPrefix(request.URL.Path, "/pings/") && request.Method == http.MethodDelete {
		handler.remove(response, request)
		return
	}
	if strings.HasSuffix(request.URL.Path, "/results") && request.Method == http.MethodGet {
		handler.results(response, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/pings/") && request.Method == http.MethodPut {
		handler.update(response, request)
		return
	}
	writeError(response, http.StatusNotFound, "not found")
}

func (handler *Handler) list(response http.ResponseWriter, request *http.Request) {
	pings, err := handler.store.List(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not list pings")
		return
	}
	writeJSON(response, http.StatusOK, pings)
}

func (handler *Handler) add(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Name   string `json:"name"`
		IP     string `json:"ip"`
		Status string `json:"status"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(response, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.IP = strings.TrimSpace(input.IP)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = StatusOff
	}
	if input.Name == "" || len(input.Name) > 255 {
		writeError(response, http.StatusBadRequest, "name is required and must be at most 255 characters")
		return
	}
	if net.ParseIP(input.IP) == nil {
		writeError(response, http.StatusBadRequest, "ip must be a valid IPv4 or IPv6 address")
		return
	}
	if !ValidStatus(input.Status) {
		writeError(response, http.StatusBadRequest, "status must be on or off")
		return
	}

	uuid, err := NewUUID()
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not create ping")
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	item := Ping{UUID: uuid, Name: input.Name, IP: input.IP, Status: input.Status, CreatedAt: timestamp, UpdatedAt: timestamp}
	if err := handler.store.Add(request.Context(), item); err != nil {
		writeError(response, http.StatusInternalServerError, "could not create ping")
		return
	}
	writeJSON(response, http.StatusCreated, item)
}

func (handler *Handler) remove(response http.ResponseWriter, request *http.Request) {
	uuid := strings.TrimPrefix(request.URL.Path, "/pings/")
	if uuid == "" || strings.Contains(uuid, "/") {
		writeError(response, http.StatusNotFound, "not found")
		return
	}
	found, err := handler.store.Remove(request.Context(), uuid)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not remove ping")
		return
	}
	if !found {
		writeError(response, http.StatusNotFound, "ping not found")
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) update(response http.ResponseWriter, request *http.Request) {
	uuid := strings.TrimPrefix(request.URL.Path, "/pings/")
	if uuid == "" || strings.Contains(uuid, "/") {
		writeError(response, http.StatusNotFound, "not found")
		return
	}

	var input struct {
		Name   *string `json:"name"`
		IP     *string `json:"ip"`
		Status *string `json:"status"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(response, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if input.Name == nil && input.IP == nil && input.Status == nil {
		writeError(response, http.StatusBadRequest, "at least one field is required")
		return
	}
	existing, found, err := handler.store.Get(request.Context(), uuid)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not find ping")
		return
	}
	if !found {
		writeError(response, http.StatusNotFound, "ping not found")
		return
	}
	if input.Name != nil {
		existing.Name = strings.TrimSpace(*input.Name)
	}
	if input.IP != nil {
		existing.IP = strings.TrimSpace(*input.IP)
	}
	if input.Status != nil {
		existing.Status = strings.TrimSpace(*input.Status)
	}
	if existing.Name == "" || len(existing.Name) > 255 {
		writeError(response, http.StatusBadRequest, "name is required and must be at most 255 characters")
		return
	}
	if net.ParseIP(existing.IP) == nil {
		writeError(response, http.StatusBadRequest, "ip must be a valid IPv4 or IPv6 address")
		return
	}
	if !ValidStatus(existing.Status) {
		writeError(response, http.StatusBadRequest, "status must be on or off")
		return
	}

	item, found, err := handler.store.Update(request.Context(), uuid, existing.Name, existing.IP, existing.Status, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not update ping")
		return
	}
	if !found {
		writeError(response, http.StatusNotFound, "ping not found")
		return
	}
	writeJSON(response, http.StatusOK, item)
}

func (handler *Handler) results(response http.ResponseWriter, request *http.Request) {
	path := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/pings/"), "/results")
	if path == "" || strings.Contains(path, "/") {
		writeError(response, http.StatusNotFound, "not found")
		return
	}
	_, found, err := handler.store.Get(request.Context(), path)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not find ping")
		return
	}
	if !found {
		writeError(response, http.StatusNotFound, "ping not found")
		return
	}
	results, err := handler.store.ListRecentResults(request.Context(), path, 60)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "could not list ping results")
		return
	}
	writeJSON(response, http.StatusOK, results)
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (response *loggingResponseWriter) WriteHeader(status int) {
	response.status = status
	response.ResponseWriter.WriteHeader(status)
}

func (response *loggingResponseWriter) Write(body []byte) (int, error) {
	if response.status == 0 {
		response.status = http.StatusOK
	}
	return response.ResponseWriter.Write(body)
}

func writeError(response http.ResponseWriter, status int, message string) {
	writeJSON(response, status, map[string]string{"error": message})
}
