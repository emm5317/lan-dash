package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"

	"github.com/emm5317/lan-dash/internal/store"
)

type DB struct {
	db   *sql.DB
	stmt *sql.Stmt
}

type Snapshot struct {
	IP        string  `json:"ip"`
	RTTms     float64 `json:"rtt_ms"`
	Alive     bool    `json:"alive"`
	OpenPorts []int   `json:"open_ports"`
	Timestamp int64   `json:"timestamp"`
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, err
		}
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scans (
		id INTEGER PRIMARY KEY,
		ip TEXT NOT NULL,
		rtt_us INTEGER,
		alive BOOLEAN,
		open_ports TEXT,
		ts INTEGER NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_scans_ip_ts ON scans(ip, ts)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	stmt, err := db.Prepare(`INSERT INTO scans (ip, rtt_us, alive, open_ports, ts) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db, stmt: stmt}, nil
}

func (d *DB) Close() error {
	d.stmt.Close()
	return d.db.Close()
}

func (d *DB) Record(dev store.Device) error {
	ports, _ := json.Marshal(dev.OpenPorts)
	_, err := d.stmt.Exec(dev.IP, dev.RTT.Microseconds(), dev.Alive, string(ports), time.Now().Unix())
	return err
}

func (d *DB) History(ip string, since time.Duration) ([]Snapshot, error) {
	cutoff := time.Now().Add(-since).Unix()
	rows, err := d.db.Query(
		`SELECT ip, rtt_us, alive, open_ports, ts FROM scans WHERE ip = ? AND ts > ? ORDER BY ts`,
		ip, cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Snapshot
	for rows.Next() {
		var s Snapshot
		var rttUs int64
		var portsJSON string
		if err := rows.Scan(&s.IP, &rttUs, &s.Alive, &portsJSON, &s.Timestamp); err != nil {
			return nil, err
		}
		s.RTTms = float64(rttUs) / 1000.0
		json.Unmarshal([]byte(portsJSON), &s.OpenPorts)
		if s.OpenPorts == nil {
			s.OpenPorts = []int{}
		}
		out = append(out, s)
	}
	if out == nil {
		out = []Snapshot{}
	}
	return out, rows.Err()
}

// Listen subscribes to store events and records each upsert to SQLite.
func (d *DB) Listen(ctx context.Context, s *store.Store) {
	ch, unsub := s.Subscribe()
	defer unsub()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.Type == store.EventUpsert {
				if err := d.Record(ev.Device); err != nil {
					slog.Warn("history: write failed", "err", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
