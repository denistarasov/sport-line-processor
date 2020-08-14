package main

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

const (
	DSN                    = "root:1234@tcp(db:3306)/golang?charset=utf8"
	databasePingRetryCount = 5
)

type storage interface {
	Upload(key string, value float64)
	Get(key string) (float64, bool)
	GetKeys() map[string]struct{}
	Count() int
}

type mapStorage struct {
	m sync.RWMutex
	s map[string]float64
}

func newMapStorage() *mapStorage {
	return &mapStorage{
		m: sync.RWMutex{},
		s: make(map[string]float64),
	}
}

func (s *mapStorage) Upload(key string, value float64) {
	s.m.Lock()
	defer s.m.Unlock()
	s.s[key] = value
}

func (s *mapStorage) Get(key string) (float64, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	value, exists := s.s[key]

	return value, exists
}

func (s *mapStorage) GetKeys() map[string]struct{} {
	s.m.RLock()
	defer s.m.RUnlock()
	keys := make(map[string]struct{}, len(s.s))

	for key := range s.s {
		keys[key] = struct{}{}
	}

	return keys
}

func (s *mapStorage) Count() int {
	s.m.RLock()
	defer s.m.RUnlock()

	return len(s.s)
}

func initDB() *sql.DB {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i != databasePingRetryCount; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
		log.Info("couldn't connect to database, retrying...")
	}
	if err != nil {
		log.Fatalf("connection to database wasn't established after %d retries", databasePingRetryCount)
	}

	queries := []string{
		`DROP TABLE IF EXISTS sportlines;`,

		`CREATE TABLE sportlines (
		  sport varchar(255) NOT NULL,
		  value double precision NOT NULL,
		  PRIMARY KEY (sport)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatal(err)
		}
	}

	return db
}

type dbStorage struct {
	db *sql.DB
}

func newDBStorage() *dbStorage {
	db := initDB()

	return &dbStorage{
		db: db,
	}
}

func (s *dbStorage) Upload(key string, value float64) {
	_, exists := s.Get(key)

	var err error

	if exists {
		_, err = s.db.Exec(
			"UPDATE sportlines SET value = ? WHERE sport = ?",
			value,
			key,
		)
	} else {
		_, err = s.db.Exec(
			"INSERT INTO sportlines (sport, value) VALUES (?, ?)",
			key,
			value,
		)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func (s *dbStorage) Get(key string) (float64, bool) {
	row := s.db.QueryRow("SELECT value FROM sportlines WHERE sport = ?", key)

	var sportLine float64

	err := row.Scan(&sportLine)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		log.Fatal(err)
	}

	return sportLine, true
}

func (s *dbStorage) GetKeys() map[string]struct{} {
	keys := make(map[string]struct{})

	rows, err := s.db.Query("SELECT sport FROM sportlines")
	if err != nil {
		log.Fatal(err)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var sport string

	for rows.Next() {
		err = rows.Scan(&sport)
		if err != nil {
			log.Fatal(err)
		}

		keys[sport] = struct{}{}
	}

	return keys
}

func (s *dbStorage) Count() int {
	row := s.db.QueryRow("SELECT COUNT(*) FROM sportlines")

	var count int

	err := row.Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	return count
}
