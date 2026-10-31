package api

import (
	"github.com/1325ok/Beszel-View-TUI/internal/model"
	"testing"
)

func TestDecodeSystem(t *testing.T) {
	d, e := decode([]byte(`{"id":"a","name":"web","status":"up","updated":"2026-10-07T01:02:03Z","info":{"cpu":41,"mp":56,"dp":37,"u":86400,"rb":1048576}}`))
	if e != nil {
		t.Fatal(e)
	}
	if d.Name != "web" || d.Status != model.Online || d.CPU != 41 || d.RAM != 56 || d.Uptime.Hours() != 24 {
		t.Fatalf("bad decode: %+v", d)
	}
}

func TestDecodeWarningStatus(t *testing.T) {
	d, err := decode([]byte(`{"id":"a","name":"nas","status":"warning","info":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != model.Warning {
		t.Fatalf("got %s, want WARNING", d.Status)
	}
}
