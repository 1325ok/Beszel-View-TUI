package screen

import (
	"github.com/1325ok/Beszel-View-TUI/internal/model"
	"testing"
	"time"
)

func TestCycleAddsSingleAlertAfterDevices(t *testing.T) {
	now := time.Unix(0, 0)
	c := New(now, time.Second, 2*time.Second)
	s := model.Snapshot{HubOnline: true, Devices: []model.Device{{Status: model.Online}, {Status: model.Down}}}
	want := []Kind{Overview, Detail, Detail, Alert, Overview}
	for i, k := range want {
		if i > 0 {
			now = now.Add(time.Second)
			if k == Overview {
				now = now.Add(time.Second)
			}
		}
		if got := c.Current(now, s).Kind; got != k {
			t.Fatalf("step %d: got %v want %v", i, got, k)
		}
	}
}
func TestNoAlertWhenHealthy(t *testing.T) {
	now := time.Unix(0, 0)
	c := New(now, time.Second, time.Second)
	s := model.Snapshot{HubOnline: true, Devices: []model.Device{{Status: model.Online}}}
	_ = c.Current(now, s)
	if got := c.Current(now.Add(time.Second), s).Kind; got != Detail {
		t.Fatal(got)
	}
	if got := c.Current(now.Add(2*time.Second), s).Kind; got != Overview {
		t.Fatal(got)
	}
}
