package common

import (
	"database/sql"
	"sync"
)

// 数据库操作
type Database struct {
	sync.RWMutex
	db *sql.DB
}

func NewDatabase(name string) (*Database, error) {
	dbName := "./" + name + ".db"
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kv (key TEXT PRIMARY KEY, value TEXT)`)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (db *Database) Set(key string, value NodeMessage) error {
	db.Lock()
	defer db.Unlock()
	_, err := db.db.Exec(`INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)`, key, value)
	return err
}

func (db *Database) Get(key string) (NodeMessage, error) {
	db.RLock()
	defer db.RUnlock()
	var value NodeMessage
	err := db.db.QueryRow(`SELECT value FROM kv WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return NodeMessage{}, err
	}
	return value, nil
}

func (db *Database) GetAll() (map[string]NodeMessage, error) {
	db.RLock()
	defer db.RUnlock()
	rows, err := db.db.Query(`SELECT key, value FROM kv`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := make(map[string]NodeMessage)
	for rows.Next() {
		var key string
		var value NodeMessage
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		data[key] = value
	}
	return data, nil
}

func (db *Database) DB() *sql.DB {
	return db.db
}
