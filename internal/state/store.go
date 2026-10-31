package state

import (
	"sync"
	"time"

	"github.com/1325ok/Beszel-View-TUI/internal/config"
	"github.com/1325ok/Beszel-View-TUI/internal/model"
)

type Store struct {
	mu      sync.RWMutex
	snap    model.Snapshot
	history map[string][2][]float64
	cfg     config.Config
}

func New(c config.Config) *Store { return &Store{cfg: c, history: map[string][2][]float64{}} }
func (s *Store) Success(ds []model.Device, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range ds {
		d := &ds[i]
		if d.Status == model.Online {
			d.Critical = d.CPU >= s.cfg.CPU.Critical || d.RAM >= s.cfg.RAM.Critical || d.Disk >= s.cfg.Disk.Critical
			if d.Critical || d.CPU >= s.cfg.CPU.Warning || d.RAM >= s.cfg.RAM.Warning || d.Disk >= s.cfg.Disk.Warning {
				d.Status = model.Warning
			}
		}
		h := s.history[d.ID]
		h[0] = appendLimit(h[0], d.CPU, 60)
		h[1] = appendLimit(h[1], d.RAM, 60)
		s.history[d.ID] = h
		d.CPUHistory = append([]float64(nil), h[0]...)
		d.RAMHistory = append([]float64(nil), h[1]...)
	}
	s.snap = model.Snapshot{Devices: ds, Updated: now, HubOnline: true}
}
func (s *Store) Failure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.HubOnline = false
	s.snap.LastError = err.Error()
}
func (s *Store) Get() model.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.snap
	v.Devices = append([]model.Device(nil), v.Devices...)
	return v
}
func appendLimit(v []float64, x float64, n int) []float64 {
	v = append(v, x)
	if len(v) > n {
		v = append([]float64(nil), v[len(v)-n:]...)
	}
	return v
}
