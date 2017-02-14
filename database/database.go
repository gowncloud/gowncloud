package db

import (
	"database/sql"
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"
	// postgres driver
	_ "github.com/lib/pq"
)

var (
	db          *sql.DB
	initialized bool

	ErrDB = errors.New("Internal database error")
)

// Connect opens a database connection
func Connect(driver, databaseurl string) {
	var err error

	if db != nil {
		log.Debug("Already connected to the database")
		return
	}

	for {
		db, err = sql.Open(driver, databaseurl)
		if err != nil {
			log.Error("Failed to connect to database: ", err)
		}
		err = db.Ping()
		if err == nil {
			break
		}
		log.Errorln("Failed to open database,", err, "- retry in 5 seconds")
		time.Sleep(5 * time.Second)
	}

	log.Info("Connected to database")
}

func Initialize() {
	if db == nil {
		log.Error("Not connected to database")
		return
	}
	if initialized {
		log.Debug("Database already initialized")
		return
	}

	log.Info("Initializing database")
	_, err := db.Exec("CREATE DATABASE IF NOT EXISTS gowncloud")
	if err != nil {
		log.Fatal("Failed to create gowncloud database: ", err)
	}
	// init settings first because they might be required to provide default values
	initSettings()

	// init tables without foreign keys
	initUsers()
	initNodes()
	// init tables with foreign keys to tables without foreign keys
	initMemberShares()
	initTrashNodes()

	initialized = true
	log.Info("Database initialized")
}

// Close closes the database connection
func Close() {
	if db == nil {
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			log.Warn("recovering from error while closing database: ", r)
		}
	}()

	db.Close()
}
