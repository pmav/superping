package ping_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/superping/internal/ping"
	"github.com/example/superping/internal/storage"
	_ "modernc.org/sqlite"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { database.Close() })
	store := storage.NewStore(database)
	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return ping.NewHandler(store)
}

func TestCreateListAndRemovePing(t *testing.T) {
	handler := testHandler(t)
	createRequest := httptest.NewRequest(http.MethodPost, "/pings", strings.NewReader(`{"name":"router","ip":"192.168.1.1","status":"on"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createResponse.Code, http.StatusCreated)
	}
	var created ping.Ping
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.UUID == "" {
		t.Fatal("created ping has no uuid")
	}
	if created.Status != ping.StatusOn {
		t.Fatalf("created status = %q, want %q", created.Status, ping.StatusOn)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" || created.CreatedAt != created.UpdatedAt {
		t.Fatalf("created timestamps = %q and %q, want equal non-empty values", created.CreatedAt, created.UpdatedAt)
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/pings/"+created.UUID, strings.NewReader(`{"name":"switch","ip":"192.168.1.2","status":"off"}`))
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d", updateResponse.Code, http.StatusOK)
	}
	var updated ping.Ping
	if err := json.NewDecoder(updateResponse.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.UUID != created.UUID || updated.CreatedAt != created.CreatedAt || updated.Name != "switch" || updated.IP != "192.168.1.2" || updated.Status != ping.StatusOff {
		t.Fatalf("updated ping = %#v", updated)
	}
	if updated.UpdatedAt == created.UpdatedAt {
		t.Fatal("update did not refresh updated_at")
	}

	partialRequest := httptest.NewRequest(http.MethodPut, "/pings/"+created.UUID, strings.NewReader(`{"status":"on"}`))
	partialResponse := httptest.NewRecorder()
	handler.ServeHTTP(partialResponse, partialRequest)
	if partialResponse.Code != http.StatusOK {
		t.Fatalf("partial update status = %d, want %d", partialResponse.Code, http.StatusOK)
	}
	var partiallyUpdated ping.Ping
	if err := json.NewDecoder(partialResponse.Body).Decode(&partiallyUpdated); err != nil {
		t.Fatal(err)
	}
	if partiallyUpdated.Name != updated.Name || partiallyUpdated.IP != updated.IP || partiallyUpdated.Status != ping.StatusOn {
		t.Fatalf("partial update = %#v, want existing name/ip with status on", partiallyUpdated)
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/pings", nil))
	var listed []ping.Ping
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].UUID != partiallyUpdated.UUID || listed[0].Name != partiallyUpdated.Name || listed[0].IP != partiallyUpdated.IP || listed[0].Status != partiallyUpdated.Status || listed[0].TotalResults != 0 {
		t.Fatalf("listed pings = %#v, want a single ping with zero total_results", listed)
	}

	resultsResponse := httptest.NewRecorder()
	handler.ServeHTTP(resultsResponse, httptest.NewRequest(http.MethodGet, "/pings/"+created.UUID+"/results", nil))
	if resultsResponse.Code != http.StatusOK {
		t.Fatalf("results status = %d, want %d", resultsResponse.Code, http.StatusOK)
	}
	aggregationsResponse := httptest.NewRecorder()
	handler.ServeHTTP(aggregationsResponse, httptest.NewRequest(http.MethodGet, "/pings/"+created.UUID+"/aggregations", nil))
	if aggregationsResponse.Code != http.StatusOK {
		t.Fatalf("aggregations status = %d, want %d", aggregationsResponse.Code, http.StatusOK)
	}
	var aggregations []ping.PingResultAggregation
	if err := json.NewDecoder(aggregationsResponse.Body).Decode(&aggregations); err != nil {
		t.Fatal(err)
	}
	if len(aggregations) != 0 {
		t.Fatalf("aggregations = %#v, want empty list", aggregations)
	}

	removeResponse := httptest.NewRecorder()
	handler.ServeHTTP(removeResponse, httptest.NewRequest(http.MethodDelete, "/pings/"+updated.UUID, nil))
	if removeResponse.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d, want %d", removeResponse.Code, http.StatusNoContent)
	}
}

func TestCreateRejectsInvalidIP(t *testing.T) {
	handler := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/pings", strings.NewReader(`{"name":"router","ip":"invalid"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateRejectsInvalidStatus(t *testing.T) {
	handler := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/pings", strings.NewReader(`{"name":"router","ip":"192.168.1.1","status":"unknown"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
