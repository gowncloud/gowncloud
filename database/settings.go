package db

import (
	"database/sql"

	log "github.com/Sirupsen/logrus"
)

// globalsettings represents the global program settings from the database
type globalsettings struct {
	// defaultAllowedSpace is the default storage space allowed for a user
	// the 0 value means there is no limit
	defaultAllowedSpace int
}

// settings represents the global settings of the program
var settings *globalsettings

// initSettings initializes the settings table
func initSettings() {
	settings = &globalsettings{}
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.settings (" +
		"defaultallowedspace INT " +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'settings': ", err)
	}

	row := db.QueryRow("SELECT * FROM gowncloud.settings")
	err = row.Scan(&settings)
	if err == sql.ErrNoRows {
		log.Warn("No settings found")
		makeDefaultSettings()
	}

	log.Debug("initialized 'settings' table")
}

func GetDefaultAllowedSpace() int {
	return settings.defaultAllowedSpace
}

// updateSettings stores the updated settings in the database table
func updateSettings() {
	// TODO: implement
}

// makeDefaultSettings generates the default settings and stores them in the database.
func makeDefaultSettings() {
	log.Warn("Generating default settings")

	settings.defaultAllowedSpace = 0

	_, err := db.Exec("INSERT INTO gowncloud.settings (defaultallowedspace) VALUES ($1)",
		settings.defaultAllowedSpace)

	if err != nil {
		log.Warn(err)
	}
}
