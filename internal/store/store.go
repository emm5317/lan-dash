package store

import (
	"strings"
	"sync"
	"time"
)

type Device struct {
	IP        string
	MAC       string
	Vendor    string
	Hostname  string
	RTT       time.Duration
	OpenPorts []int
	Alive     bool
	LastSeen  time.Time
	Speed     string
}

func (d Device) RTTms() float64 {
	return float64(d.RTT.Microseconds()) / 1000.0
}

func (d Device) SafeID() string {
	// replace dots for use as HTML id attribute
	r := strings.NewReplacer(".", "-")
	return "row-" + r.Replace(d.IP)
}

// DeviceGroup represents a logical grouping of devices on the network.
type DeviceGroup string

const (
	GroupNetwork  DeviceGroup = "Network"
	GroupComputer DeviceGroup = "Computers"
	GroupIoT      DeviceGroup = "IoT"
	GroupUnknown  DeviceGroup = "Unknown"
)

// GroupOrder defines the display order for device groups.
var GroupOrder = map[DeviceGroup]int{
	GroupNetwork:  0,
	GroupComputer: 1,
	GroupIoT:      2,
	GroupUnknown:  3,
}

// ClassifyDevice determines the DeviceGroup for a device based on vendor and hostname.
func ClassifyDevice(vendor, hostname string) DeviceGroup {
	v := strings.ToLower(vendor)
	h := strings.ToLower(hostname)

	networkKeywords := []string{"cisco", "netgear", "tp-link", "ubiquiti", "aruba", "mikrotik", "router", "gateway"}
	for _, kw := range networkKeywords {
		if strings.Contains(v, kw) || strings.Contains(h, kw) {
			return GroupNetwork
		}
	}

	computerKeywords := []string{"apple", "dell", "lenovo", "hp", "asus", "microsoft", "intel"}
	for _, kw := range computerKeywords {
		if strings.Contains(v, kw) || strings.Contains(h, kw) {
			return GroupComputer
		}
	}

	iotKeywords := []string{"sonos", "philips", "ring", "nest", "samsung", "lg", "roku", "amazon", "espressif", "raspberry"}
	for _, kw := range iotKeywords {
		if strings.Contains(v, kw) || strings.Contains(h, kw) {
			return GroupIoT
		}
	}

	return GroupUnknown
}

// Group returns the DeviceGroup this device belongs to.
func (d Device) Group() DeviceGroup {
	return ClassifyDevice(d.Vendor, d.Hostname)
}

type EventType int

const (
	EventUpsert EventType = iota
	EventOffline
)

type Event struct {
	Type   EventType
	Device Device
}

type Store struct {
	mu      sync.RWMutex
	devices map[string]*Device
	subs    []chan Event
}

func New() *Store {
	return &Store{devices: make(map[string]*Device)}
}

func (s *Store) Upsert(d Device) Device {
	d.LastSeen = time.Now()
	s.mu.Lock()
	existing, ok := s.devices[d.IP]
	if ok {
		// preserve enriched fields that arrive asynchronously
		if d.MAC == "" {
			d.MAC = existing.MAC
		}
		if d.Vendor == "" {
			d.Vendor = existing.Vendor
		}
		if d.Hostname == "" {
			d.Hostname = existing.Hostname
		}
		if len(d.OpenPorts) == 0 {
			d.OpenPorts = existing.OpenPorts
		}
		if d.Speed == "" {
			d.Speed = existing.Speed
		}
	}
	s.devices[d.IP] = &d
	subs := make([]chan Event, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	ev := Event{Type: EventUpsert, Device: d}
	for _, ch := range subs {
		select {
		case ch <- ev:
		default: // slow consumer — drop, never block the scanner goroutine
		}
	}
	return d
}

func (s *Store) SetAlive(ip string, alive bool) {
	s.mu.Lock()
	d, ok := s.devices[ip]
	if !ok {
		s.mu.Unlock()
		return
	}
	d.Alive = alive
	subs := make([]chan Event, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	ev := Event{Type: EventOffline, Device: *d}
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (s *Store) All() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, *d)
	}
	return out
}

func (s *Store) Subscribe() (chan Event, func()) {
	ch := make(chan Event, 16)
	s.mu.Lock()
	s.subs = append(s.subs, ch)
	s.mu.Unlock()
	return ch, func() { s.unsubscribe(ch) }
}

func (s *Store) unsubscribe(ch chan Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.subs {
		if sub == ch {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)
			close(ch)
			return
		}
	}
}
