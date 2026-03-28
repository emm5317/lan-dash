package store

import (
	"sync"
	"time"
)

type Device struct {
	IP        string
	MAC       string
	RTT       time.Duration
	OpenPorts []int
	LastSeen  time.Time
}

type Store struct {
	mu      sync.RWMutex
	devices map[string]*Device
	events  chan DeviceEvent
}

type DeviceEvent struct {
	Type   string // "add", "update", "remove"
	Device *Device
}

func NewStore() *Store {
	return &Store{
		devices: make(map[string]*Device),
		events:  make(chan DeviceEvent, 100),
	}
}

func (s *Store) GetDevices() map[string]*Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	devices := make(map[string]*Device)
	for k, v := range s.devices {
		devices[k] = v
	}
	return devices
}

func (s *Store) UpdateDeviceFromARP(ip, mac string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, exists := s.devices[ip]; exists {
		d.LastSeen = time.Now()
	} else {
		s.devices[ip] = &Device{IP: ip, MAC: mac, LastSeen: time.Now()}
		s.events <- DeviceEvent{Type: "add", Device: s.devices[ip]}
	}
}

func (s *Store) UpdateDeviceRTT(ip string, rtt time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, exists := s.devices[ip]; exists {
		d.RTT = rtt
		s.events <- DeviceEvent{Type: "update", Device: d}
	}
}

func (s *Store) AddOpenPort(ip string, port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, exists := s.devices[ip]; exists {
		d.OpenPorts = append(d.OpenPorts, port)
		s.events <- DeviceEvent{Type: "update", Device: d}
	}
}

func (s *Store) Events() <-chan DeviceEvent {
	return s.events
}
