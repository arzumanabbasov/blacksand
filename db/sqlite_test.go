package db

import (
	"path/filepath"
	"testing"
)

func TestSQLiteOpenPersistsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blacksand.db")
	database, err := Open("sqlite:" + path)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.SetSetting("sqlite-test", "persisted"); err != nil {
		t.Fatal(err)
	}
	agents, err := database.ListAgents()
	if err != nil || len(agents) < 6 {
		t.Fatalf("builtin agents=%d err=%v", len(agents), err)
	}
	database.Close()
	database, err = Open("sqlite:" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	value, found, err := database.GetSetting("sqlite-test")
	if err != nil || !found || value != "persisted" {
		t.Fatalf("settings not persisted: %q %v %v", value, found, err)
	}
}
