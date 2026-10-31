package model

import "time"

type Status int

const (
	Unknown Status = iota
	Online
	Warning
	Down
)

func (s Status) String() string { return [...]string{"UNKNOWN", "ONLINE", "WARNING", "DOWN"}[s] }

type Device struct {
	ID, Name, Description                        string
	Status                                       Status
	CPU, RAM, Disk, Load, Temp, RX, TX           float64
	MemoryUsed, MemoryTotal, DiskUsed, DiskTotal float64
	Critical                                     bool
	Uptime                                       time.Duration
	Updated                                      time.Time
	CPUHistory, RAMHistory                       []float64
}
type Snapshot struct {
	Devices   []Device
	Updated   time.Time
	HubOnline bool
	LastError string
}
type Counts struct{ Up, Warn, Down, Unknown int }

func (s Snapshot) Counts() (c Counts) {
	for _, d := range s.Devices {
		switch d.Status {
		case Online:
			c.Up++
		case Warning:
			c.Warn++
		case Down:
			c.Down++
		default:
			c.Unknown++
		}
	}
	return
}
