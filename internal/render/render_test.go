package render

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/1325ok/Beszel-View-TUI/internal/config"
	"github.com/1325ok/Beszel-View-TUI/internal/model"
	"github.com/1325ok/Beszel-View-TUI/internal/screen"
)

func TestSmallScreensUseSameFixedDimensions(t *testing.T) {
	now := time.Unix(1000, 0)
	cfg := config.Config{Layout: "small", AlertDuration: 20 * time.Second, AlertBlink: time.Second}
	r := New(cfg)
	s := model.Snapshot{
		HubOnline: true,
		Updated:   now,
		Devices: []model.Device{
			{Name: "server-up", Status: model.Online, CPU: 20, RAM: 30, Disk: 40},
			{Name: "server-down", Status: model.Down, Updated: now.Add(-time.Minute)},
		},
	}
	views := []screen.View{
		{Kind: screen.Overview, Started: now},
		{Kind: screen.Detail, Index: 0, Started: now},
		{Kind: screen.Alert, Started: now},
	}
	for _, height := range []int{13, 16, 30, 50} {
		for _, view := range views {
			out := r.small(view, s, 50, height, now)
			assertBoxDimensions(t, out, 50, height)
			rows := strings.Split(out, "\r\n")
			if rows[len(rows)-2] == "│"+strings.Repeat(" ", 48)+"│" {
				t.Fatalf("height %d view %v leaves all spacing at the bottom", height, view.Kind)
			}
		}
	}
}

func TestSmallOverviewKeepsGaugeRowsSeparated(t *testing.T) {
	now := time.Unix(1000, 0)
	r := New(config.Config{Layout: "small", AlertDuration: 20 * time.Second, AlertBlink: time.Second})
	s := model.Snapshot{HubOnline: true, Devices: []model.Device{{Name: "server", Status: model.Online, CPU: 33, RAM: 90, Disk: 59}}}
	for _, height := range []int{13, 16, 30} {
		rows := strings.Split(r.small(screen.View{Kind: screen.Overview}, s, 50, height, now), "\r\n")
		cpuRow, ramRow, diskRow := rowContaining(rows, "CPU"), rowContaining(rows, "RAM"), rowContaining(rows, "DISK")
		if cpuRow < 0 || ramRow-cpuRow < 2 || diskRow-ramRow < 2 {
			t.Fatalf("height %d gauge rows are not separated: CPU=%d RAM=%d DISK=%d", height, cpuRow, ramRow, diskRow)
		}
	}
}

func rowContaining(rows []string, text string) int {
	for i, row := range rows {
		if strings.Contains(row, text) {
			return i
		}
	}
	return -1
}

func TestSpreadRowsUsesAvailableHeight(t *testing.T) {
	input := []string{"top", "middle", "bottom"}
	got := spreadRows(input, 7)
	wantRows := map[int]string{0: "top", 3: "middle", 6: "bottom"}
	for row, want := range wantRows {
		if got[row] != want {
			t.Fatalf("row %d = %q, want %q; all rows: %#v", row, got[row], want, got)
		}
	}
}

func TestSingleModeShowsAllDevicesWithoutDetailScreen(t *testing.T) {
	now := time.Unix(1000, 0)
	r := New(config.Config{Layout: "small", SingleMode: true, AlertDuration: 20 * time.Second, AlertBlink: time.Second})
	s := model.Snapshot{HubOnline: true, Updated: now, Devices: []model.Device{
		{Name: "alpha", Status: model.Online},
		{Name: "beta", Status: model.Warning},
		{Name: "gamma", Status: model.Down},
	}}
	out := r.small(screen.View{Kind: screen.Detail, Index: 2}, s, 60, 20, now)
	if strings.Contains(out, "SINGLE") {
		t.Fatal("single mode should not add a redundant SINGLE label")
	}
	for _, want := range []string{"BESZEL", "alpha", "beta", "gamma"} {
		if !strings.Contains(out, want) {
			t.Fatalf("single mode output does not contain %q", want)
		}
	}
}

func TestSingleTableColumnsAlign(t *testing.T) {
	d := model.Device{Name: "LocalTest", Status: model.Online, CPU: 11, RAM: 22, Disk: 33}
	for _, tc := range []struct {
		width int
		large bool
	}{{50, false}, {120, true}} {
		headerText := stripANSI(singleTableHeader(tc.width, tc.large))
		rowText := stripANSI(singleDeviceRow(d, tc.width, tc.large))
		for label, value := range map[string]string{"CPU": "11%", "RAM": "22%", "DSK": "33%", "STATUS": "ONLINE"} {
			if cellColumn(headerText, label) != cellColumn(rowText, value) {
				t.Fatalf("width %d column %s is misaligned\nheader: %q\nrow:    %q", tc.width, label, headerText, rowText)
			}
		}
	}
}

func TestLargeDetailFitsMinimumSize(t *testing.T) {
	now := time.Unix(1000, 0)
	r := New(config.Config{Layout: "large", AlertDuration: 20 * time.Second, AlertBlink: time.Second})
	s := model.Snapshot{HubOnline: true, Devices: []model.Device{{Name: "server", Status: model.Online, CPU: 30, RAM: 50, Disk: 70}}}
	out := r.large(screen.View{Kind: screen.Detail, Index: 0}, s, 80, 22, now)
	assertBoxDimensions(t, out, 80, 22)
}

func cellColumn(s, needle string) int {
	index := strings.Index(s, needle)
	if index < 0 {
		return -1
	}
	return visible(s[:index])
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == 0x1b {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestSingleModeReportsHiddenDevices(t *testing.T) {
	r := New(config.Config{Layout: "small", SingleMode: true, AlertDuration: 20 * time.Second, AlertBlink: time.Second})
	s := model.Snapshot{HubOnline: true}
	for i := 0; i < 20; i++ {
		s.Devices = append(s.Devices, model.Device{Name: fmt.Sprintf("server-%02d", i), Status: model.Online})
	}
	out := r.small(screen.View{Kind: screen.Overview}, s, 50, 13, time.Unix(1000, 0))
	if !strings.Contains(out, "MORE") {
		t.Fatal("single mode must report devices that do not fit")
	}
}

func TestConstrainedAlertKeepsCountdown(t *testing.T) {
	now := time.Unix(1000, 0)
	r := New(config.Config{Layout: "small", AlertDuration: 20 * time.Second, AlertBlink: time.Second})
	s := model.Snapshot{HubOnline: true}
	for i := 0; i < 10; i++ {
		s.Devices = append(s.Devices, model.Device{Name: "down", Status: model.Down})
	}
	out := r.small(screen.View{Kind: screen.Alert, Started: now}, s, 38, 16, now)
	assertBoxDimensions(t, out, 38, 16)
	if !strings.Contains(out, "Monitoring continues") {
		t.Fatal("countdown row was removed from constrained alert")
	}
}

func TestFitPreservesANSIAndWideCharacters(t *testing.T) {
	input := green + "서버-대한민국-ONLINE" + reset
	got := fit(input, 12) // ten visible cells inside the border
	if width := visible(got); width > 10 {
		t.Fatalf("fitted text is %d cells wide, want at most 10: %q", width, got)
	}
	if !strings.HasSuffix(got, reset) {
		t.Fatalf("fitted colored text must reset terminal style: %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("fit produced invalid UTF-8: %q", got)
	}
}

func assertBoxDimensions(t *testing.T, out string, width, height int) {
	t.Helper()
	rows := strings.Split(out, "\r\n")
	if len(rows) != height {
		t.Fatalf("got %d rows, want %d", len(rows), height)
	}
	for i, row := range rows {
		if got := visible(row); got != width {
			t.Fatalf("row %d has width %d, want %d: %q", i, got, width, row)
		}
	}
}
