package monitor

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/example/superping/internal/ping"
	"github.com/example/superping/internal/storage"
	_ "modernc.org/sqlite"
)

func TestCheckAllStoresSuccessAndFailureResults(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	defer database.Close()

	store := storage.NewStore(database)
	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	items := []ping.Ping{
		{UUID: "success-id", Name: "reachable", IP: "192.0.2.1", Status: ping.StatusOn, CreatedAt: "now", UpdatedAt: "now"},
		{UUID: "failure-id", Name: "failed", IP: "192.0.2.2", Status: ping.StatusOn, CreatedAt: "now", UpdatedAt: "now"},
		{UUID: "off-id", Name: "disabled", IP: "192.0.2.3", Status: ping.StatusOff, CreatedAt: "now", UpdatedAt: "now"},
	}
	for _, item := range items {
		if err := store.Add(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}

	monitor := New(store)
	monitor.runner = func(_ context.Context, address string) error {
		if address == "192.0.2.2" {
			return errors.New("timeout")
		}
		return nil
	}
	monitor.checkAll(context.Background())

	for _, want := range []struct {
		uuid, status, message string
	}{
		{"success-id", ping.ResultSuccess, ""},
		{"failure-id", ping.ResultFailed, "timeout"},
	} {
		results, err := store.ListResults(context.Background(), want.uuid)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Status != want.status || results[0].Message != want.message || results[0].Duration < 0 || results[0].Timestamp == "" {
			t.Fatalf("results for %s = %#v", want.uuid, results)
		}
	}
	offResults, err := store.ListResults(context.Background(), "off-id")
	if err != nil {
		t.Fatal(err)
	}
	if len(offResults) != 0 {
		t.Fatalf("off ping results = %#v, want none", offResults)
	}
}