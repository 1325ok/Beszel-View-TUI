package render

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/1325ok/Beszel-View-TUI/internal/config"
	"github.com/1325ok/Beszel-View-TUI/internal/model"
	"github.com/1325ok/Beszel-View-TUI/internal/screen"
)

const (
	reset  = "\x1b[0m"
	red    = "\x1b[38;2;255;65;65m"
	green  = "\x1b[38;2;70;220;130m"
	yellow = "\x1b[38;2;255;200;50m"
	cyan   = "\x1b[38;2;60;200;220m"
	gray   = "\x1b[38;2;150;160;170m"
	white  = "\x1b[38;2;235;240;245m"
	redBG  = "\x1b[48;2;150;0;0m\x1b[38;2;255;255;255m"
)

type Renderer struct{ cfg config.Config }

func New(c config.Config) *Renderer { return &Renderer{cfg: c} }
func (r *Renderer) Render(v screen.View, s model.Snapshot, w, h int, now time.Time) string {
	layout := r.cfg.Layout
	if layout == "auto" {
		if w >= 100 && h >= 30 {
			layout = "large"
		} else {
			layout = "small"
		}
	}
	var body string
	if layout == "large" {
		body = r.large(v, s, w, h, now)
	} else {
		body = r.small(v, s, w, h, now)
	}
	return "\x1b[H\x1b[2J\x1b[?25l" + body + reset
}
func (r *Renderer) small(v screen.View, s model.Snapshot, w, h int, now time.Time) string {
	// Small displays commonly use 80 columns. Keep a reasonable upper bound,
	// but do not shrink an explicitly selected 80x24 viewport to 72 columns.
	w = clamp(w, 38, 96)
	if r.cfg.SingleMode {
		return r.smallSingle(s, w, h, now)
	}
	switch v.Kind {
	case screen.Detail:
		if v.Index < len(s.Devices) {
			return r.smallDetail(s.Devices[v.Index], v.Index, len(s.Devices), w, h, now)
		}
	case screen.Alert:
		return r.alert(s, w, h, now, v.Started)
	}
	return r.smallOverview(s, w, h, now)
}

func (r *Renderer) smallSingle(s model.Snapshot, w, h int, now time.Time) string {
	c := s.Counts()
	lines := []string{
		header("BESZEL", hub(s), w),
		rule(w),
		center(fmt.Sprintf("%s● %d UP%s  %s▲ %d WARN%s  %s✕ %d DOWN%s", green, c.Up, reset, yellow, c.Warn, reset, red, c.Down, reset), w),
		singleTableHeader(w, false),
		rule(w),
	}
	footer := fit(" Updated "+timeText(s.Updated), w)
	if !s.HubOnline {
		footer = fit(" "+red+"HUB OFFLINE"+reset+"  "+stale(s, now), w)
	}
	appendDeviceRows(&lines, s.Devices, max(0, h-2-len(lines)-1), w, false)
	lines = append(lines, footer)
	return box(lines, w, h)
}
func (r *Renderer) smallOverview(s model.Snapshot, w, h int, now time.Time) string {
	c := s.Counts()
	cpu, ram, disk, _, _ := averages(s)
	lines := []string{
		header("BESZEL", hub(s), w),
		rule(w),
		center(fmt.Sprintf("DEVICES  %d", len(s.Devices)), w),
		center(fmt.Sprintf("%s● %d UP%s    %s✕ %d DOWN%s", green, c.Up, reset, red, c.Down, reset), w),
	}
	// Preserve separation between all three gauges even at the 13-row minimum.
	// Taller screens get an additional gap between the summary and gauges.
	if h > 13 {
		lines = append(lines, "")
	}
	lines = append(lines,
		metric("CPU", cpu, w-4),
		"",
		metric("RAM", ram, w-4),
		"",
		metric("DISK", disk, w-4),
		rule(w),
	)
	if !s.HubOnline {
		lines = append(lines, center(red+"HUB CONNECTION LOST"+reset, w), center(stale(s, now), w))
	} else if c.Down+c.Warn == 0 {
		lines = append(lines, center(green+"✓ ALL SYSTEMS OPERATIONAL"+reset, w))
	} else {
		d := firstAlert(s)
		lines = append(lines, fit(fmt.Sprintf("%s✕ %-18s %-7s%s %s", red, d.Name, d.Status, reset, elapsed(d.Updated, now)), w))
	}
	return box(lines, w, h)
}
func (r *Renderer) smallDetail(d model.Device, i, n, w, h int, now time.Time) string {
	lines := []string{header(fmt.Sprintf("%02d/%02d  %s", i+1, n, d.Name), deviceStatus(d), w), rule(w), "", metric("CPU", d.CPU, w-4), "", metric("RAM", d.RAM, w-4), "", metric("DISK", d.Disk, w-4), "", rule(w), fit(fmt.Sprintf("↓%s ↑%s   %s   UP %s", rate(d.RX), rate(d.TX), temp(d.Temp), shortDuration(d.Uptime)), w)}
	return box(lines, w, h)
}
func (r *Renderer) large(v screen.View, s model.Snapshot, w, h int, now time.Time) string {
	w = clamp(w, 80, 160)
	if r.cfg.SingleMode {
		return r.largeSingle(s, w, h, now)
	}
	if v.Kind == screen.Alert {
		return r.alert(s, w, h, now, v.Started)
	}
	if v.Kind == screen.Detail && v.Index < len(s.Devices) {
		return r.largeDetail(s.Devices[v.Index], v.Index, len(s.Devices), w, h, now)
	}
	c := s.Counts()
	cpu, ram, disk, rx, tx := averages(s)
	lines := []string{header("BESZEL MONITOR", fmt.Sprintf("%s  %d DEVICES  %s● %d UP%s  %s▲ %d WARN%s  %s✕ %d DOWN%s", hub(s), len(s.Devices), green, c.Up, reset, yellow, c.Warn, reset, red, c.Down, reset), w), rule(w), "  SYSTEMS", singleTableHeader(w, true), rule(w)}
	maxRows := min(len(s.Devices), max(4, h-17))
	for i := 0; i < maxRows; i++ {
		d := s.Devices[i]
		lines = append(lines, singleDeviceRow(d, w, true))
	}
	lines = append(lines, "", fmt.Sprintf("  OVERALL   CPU %s   RAM %s   DISK %s", bar(cpu, 12), bar(ram, 12), bar(disk, 12)), fmt.Sprintf("  NETWORK   ↓ %s   ↑ %s", rate(rx), rate(tx)), "", fit("  CPU HISTORY  "+spark(overallHistory(s, true), max(12, w-18)), w), fit("  RAM HISTORY  "+spark(overallHistory(s, false), max(12, w-18)), w), rule(w), header("", fmt.Sprintf("Updated %s  %s", timeText(s.Updated), hub(s)), w))
	return box(lines, w, h)
}

func (r *Renderer) largeSingle(s model.Snapshot, w, h int, now time.Time) string {
	c := s.Counts()
	lines := []string{
		header("BESZEL MONITOR", hub(s), w),
		center(fmt.Sprintf("%d DEVICES   %s● %d UP%s   %s▲ %d WARN%s   %s✕ %d DOWN%s", len(s.Devices), green, c.Up, reset, yellow, c.Warn, reset, red, c.Down, reset), w),
		rule(w),
		singleTableHeader(w, true),
		rule(w),
	}
	footer := header("", "Updated "+timeText(s.Updated), w)
	if !s.HubOnline {
		footer = header(red+"HUB CONNECTION LOST"+reset, stale(s, now), w)
	}
	appendDeviceRows(&lines, s.Devices, max(0, h-2-len(lines)-2), w, true)
	lines = append(lines, rule(w), footer)
	return box(lines, w, h)
}

func appendDeviceRows(lines *[]string, devices []model.Device, slots, w int, large bool) {
	shown := min(len(devices), slots)
	if len(devices) > slots {
		shown = max(0, slots-1)
	}
	for i := 0; i < shown; i++ {
		d := devices[i]
		*lines = append(*lines, singleDeviceRow(d, w, large))
	}
	if shown < len(devices) && slots > 0 {
		*lines = append(*lines, center(fmt.Sprintf("+%d MORE — increase --size height", len(devices)-shown), w))
	}
}

func singleDeviceRow(d model.Device, w int, large bool) string {
	state := deviceStatus(d)
	metrics := singleMetrics(compactPct(d, d.CPU), compactPct(d, d.RAM), compactPct(d, d.Disk), large)
	stateWidth := 10
	nameWidth := max(1, w-2-visible(metrics)-stateWidth-2)
	name := padANSI(fit(d.Name, nameWidth+2), nameWidth)
	return fit(" "+name+metrics+" "+padLeftANSI(state, stateWidth), w)
}

func singleTableHeader(w int, large bool) string {
	metrics := singleMetrics("CPU", "RAM", "DSK", large)
	stateWidth := 10
	nameWidth := max(1, w-2-visible(metrics)-stateWidth-2)
	return fit(" "+padANSI("DEVICE", nameWidth)+metrics+" "+padLeftANSI("STATUS", stateWidth), w)
}

func singleMetrics(cpu, ram, disk string, large bool) string {
	if large {
		return fmt.Sprintf(" %5s   %5s   %5s", cpu, ram, disk)
	}
	return fmt.Sprintf(" %4s %4s %4s", cpu, ram, disk)
}

func padLeftANSI(s string, width int) string {
	return strings.Repeat(" ", max(0, width-visible(s))) + s
}

func compactPct(d model.Device, value float64) string {
	if d.Status == model.Down || d.Status == model.Unknown {
		return "--"
	}
	return fmt.Sprintf("%.0f%%", value)
}
func (r *Renderer) largeDetail(d model.Device, i, n, w, h int, now time.Time) string {
	columnWidth := max(8, (w-2)/3)
	barWidth := max(4, columnWidth-7)
	lines := []string{
		header(fmt.Sprintf("DEVICE %02d/%02d", i+1, n), "AUTO ●", w),
		header("  "+d.Name, deviceStatus(d), w),
		header("  "+d.Description, "UPTIME "+shortDuration(d.Uptime), w),
		rule(w),
		columnsLine([]string{"CPU", "MEMORY", "DISK"}, w),
		columnsLine([]string{fmt.Sprintf("%.1f%%", d.CPU), fmt.Sprintf("%.1f%%", d.RAM), fmt.Sprintf("%.1f%%", d.Disk)}, w),
		columnsLine([]string{bar(d.CPU, barWidth), bar(d.RAM, barWidth), bar(d.Disk, barWidth)}, w),
		columnsLine([]string{fmt.Sprintf("Load %.2f", d.Load), fmt.Sprintf("%.1f / %.1f GB", d.MemoryUsed, d.MemoryTotal), fmt.Sprintf("%.1f / %.1f GB", d.DiskUsed, d.DiskTotal)}, w),
		columnsLine([]string{"Temp " + temp(d.Temp), "", ""}, w),
		rule(w),
		"  CPU HISTORY",
		fit("  "+spark(d.CPUHistory, w-5), w),
		"  MEMORY HISTORY",
		fit("  "+spark(d.RAMHistory, w-5), w),
		columnsLine([]string{"NETWORK", "↓ DOWNLOAD " + rate(d.RX), "↑ UPLOAD " + rate(d.TX)}, w),
		rule(w),
		header("  Monitoring live", now.Format("15:04:05"), w),
	}
	return box(lines, w, h)
}

func columnsLine(items []string, w int) string {
	if len(items) == 0 {
		return ""
	}
	total := max(1, w-2)
	base := total / len(items)
	remainder := total % len(items)
	var b strings.Builder
	for i, item := range items {
		width := base
		if i < remainder {
			width++
		}
		item = fit(item, width+2)
		left := max(0, (width-visible(item))/2)
		right := max(0, width-visible(item)-left)
		b.WriteString(strings.Repeat(" ", left))
		b.WriteString(item)
		b.WriteString(strings.Repeat(" ", right))
	}
	return b.String()
}
func (r *Renderer) alert(s model.Snapshot, w, h int, now, start time.Time) string {
	blink := int(now.Sub(start)/r.cfg.AlertBlink)%2 == 1
	paint := red
	if blink {
		paint = redBG
	}
	c := s.Counts()
	remain := r.cfg.AlertDuration - now.Sub(start)
	if remain < 0 {
		remain = 0
	}
	title := fmt.Sprintf("!! %d ALERTS !!", c.Down+c.Warn)
	if !s.HubOnline {
		title = "!! HUB CONNECTION LOST !!"
	}
	lines := []string{center(paint+title+reset, w), rule(w), ""}
	if !s.HubOnline {
		lines = append(lines, center(paint+"HUB OFFLINE"+reset, w), center("LAST UPDATE "+timeText(s.Updated), w), center(stale(s, now), w), "")
	} else {
		shown := 0
		for _, d := range s.Devices {
			if d.Status == model.Down || d.Status == model.Warning {
				label := d.Status.String()
				if d.Critical {
					label = "CRITICAL"
				}
				lines = append(lines, fit(fmt.Sprintf("  %s%-26s %-9s%s  %s", paint, d.Name, label, reset, elapsed(d.Updated, now)), w))
				shown++
				if shown >= 8 {
					break
				}
			}
		}
		lines = append(lines, "")
	}
	lines = append(lines, center(fmt.Sprintf("%d UP / %d WARN / %d DOWN", c.Up, c.Warn, c.Down), w), "", center(fmt.Sprintf("Monitoring continues in %02d:%02d", int(remain.Minutes()), int(remain.Seconds())%60), w))
	return box(lines, w, h)
}
func box(lines []string, w, h int) string {
	contentHeight := max(1, h-2)
	if len(lines) > contentHeight {
		// Keep the final status/countdown row visible on constrained screens.
		trimmed := append([]string(nil), lines[:contentHeight-1]...)
		lines = append(trimmed, lines[len(lines)-1])
	}
	lines = spreadRows(lines, contentHeight)
	var b strings.Builder
	b.WriteString("┌" + strings.Repeat("─", w-2) + "┐\r\n")
	for _, l := range lines {
		b.WriteString("│" + padANSI(fit(l, w), w-2) + "│\r\n")
	}
	b.WriteString("└" + strings.Repeat("─", w-2) + "┘")
	return b.String()
}

// spreadRows uses every available row without dumping all unused space below
// the content. On short displays the layout stays compact; on tall displays
// the same information is distributed evenly from the top to the bottom.
func spreadRows(lines []string, height int) []string {
	if height <= 0 {
		return nil
	}
	if len(lines) >= height {
		return lines[:height]
	}
	if len(lines) == 0 {
		return make([]string, height)
	}
	if len(lines) == 1 {
		out := make([]string, height)
		out[height/2] = lines[0]
		return out
	}
	out := make([]string, height)
	for i, line := range lines {
		row := i * (height - 1) / (len(lines) - 1)
		out[row] = line
	}
	return out
}
func header(l, r string, w int) string {
	gap := w - 2 - visible(l) - visible(r)
	if gap < 1 {
		gap = 1
	}
	return l + strings.Repeat(" ", gap) + r
}
func rule(w int) string { return strings.Repeat("─", w-2) }
func center(s string, w int) string {
	n := max(0, (w-2-visible(s))/2)
	return strings.Repeat(" ", n) + s
}
func fit(s string, w int) string {
	limit := max(0, w-2)
	if visible(s) <= limit {
		return s
	}
	var b strings.Builder
	cells := 0
	hadANSI := false
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			end := strings.IndexByte(s[i:], 'm')
			if end < 0 {
				break
			}
			end += i + 1
			b.WriteString(s[i:end])
			i = end
			hadANSI = true
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		width := runeWidth(r)
		if cells+width > limit {
			break
		}
		b.WriteRune(r)
		cells += width
		i += size
	}
	if hadANSI {
		b.WriteString(reset)
	}
	return b.String()
}
func padANSI(s string, n int) string { return s + strings.Repeat(" ", max(0, n-visible(s))) }
func visible(s string) int {
	in := false
	n := 0
	for _, x := range s {
		if x == 27 {
			in = true
			continue
		}
		if in {
			if x == 'm' {
				in = false
			}
			continue
		}
		n += runeWidth(x)
	}
	return n
}

func runeWidth(r rune) int {
	if r == 0 || r < 32 || (r >= 0x7f && r < 0xa0) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	// Common East Asian wide/full-width and emoji ranges used by device names
	// and status indicators. Box-drawing and block characters remain width 1.
	if r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

func bar(v float64, n int) string {
	fill := int(math.Round(v / 100 * float64(n)))
	fill = clamp(fill, 0, n)
	color := cyan
	if v >= 90 {
		color = red
	} else if v >= 75 {
		color = yellow
	}
	return color + strings.Repeat("█", fill) + gray + strings.Repeat("░", n-fill) + reset + fmt.Sprintf(" %3.0f%%", v)
}
func metric(name string, v float64, w int) string {
	return fmt.Sprintf(" %-5s %s", name, bar(v, max(8, w-15)))
}
func status(s model.Status) string {
	switch s {
	case model.Online:
		return green + "● ONLINE" + reset
	case model.Warning:
		return yellow + "▲ WARNING" + reset
	case model.Down:
		return red + "✕ DOWN" + reset
	default:
		return gray + "? UNKNOWN" + reset
	}
}
func deviceStatus(d model.Device) string {
	if d.Critical {
		return red + "▲ CRITICAL" + reset
	}
	return status(d.Status)
}
func hub(s model.Snapshot) string {
	if s.HubOnline {
		return green + "● HUB" + reset
	}
	return red + "✕ HUB OFFLINE" + reset
}
func firstAlert(s model.Snapshot) model.Device {
	for _, d := range s.Devices {
		if d.Status == model.Down || d.Status == model.Warning {
			return d
		}
	}
	return model.Device{}
}
func averages(s model.Snapshot) (cpu, ram, disk, rx, tx float64) {
	n := 0.
	for _, d := range s.Devices {
		if d.Status != model.Down && d.Status != model.Unknown {
			cpu += d.CPU
			ram += d.RAM
			disk += d.Disk
			rx += d.RX
			tx += d.TX
			n++
		}
	}
	if n > 0 {
		cpu /= n
		ram /= n
		disk /= n
	}
	return
}
func pctOrDash(d model.Device, v float64) string {
	if d.Status == model.Down || d.Status == model.Unknown {
		return "  -- "
	}
	return fmt.Sprintf("%4.0f%%", v)
}
func elapsed(t, now time.Time) string {
	if t.IsZero() {
		return "--:--"
	}
	return shortDuration(now.Sub(t))
}
func stale(s model.Snapshot, now time.Time) string {
	if s.Updated.IsZero() {
		return "NO CACHED DATA"
	}
	return "DATA STALE " + shortDuration(now.Sub(s.Updated))
}
func shortDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd%02dh", int(d.Hours()/24), int(d.Hours())%24)
	}
	return fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}
func timeText(t time.Time) string {
	if t.IsZero() {
		return "--:--:--"
	}
	return t.Local().Format("15:04:05")
}
func rate(v float64) string {
	if v <= 0 {
		return "--"
	}
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f%s", v, units[i])
}
func temp(v float64) string {
	if v == 0 {
		return "--°C"
	}
	return fmt.Sprintf("%.0f°C", v)
}
func spark(v []float64, n int) string {
	chars := []rune("▁▂▃▄▅▆▇█")
	if len(v) > n {
		v = v[len(v)-n:]
	}
	var b strings.Builder
	for _, x := range v {
		i := clamp(int(x/100*7), 0, 7)
		b.WriteRune(chars[i])
	}
	if len(v) == 0 {
		return gray + "waiting for samples" + reset
	}
	return cyan + b.String() + reset
}
func overallHistory(s model.Snapshot, cpu bool) []float64 {
	var best []float64
	for _, d := range s.Devices {
		v := d.RAMHistory
		if cpu {
			v = d.CPUHistory
		}
		if len(v) > len(best) {
			best = v
		}
	}
	return best
}
func clamp(v, a, b int) int {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
