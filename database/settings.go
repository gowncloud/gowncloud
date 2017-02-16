package db

import log "github.com/Sirupsen/logrus"

var (
	// DEFAULT_ALLOWED_SPACE is the default allowed space, assigned to every new user
	DEFAULT_ALLOWED_SPACE = "defaultallowedspace"
	// DAV_ROOT is the root directory of the dav server.
	DAV_ROOT = "davroot"
	// VERSION is the current version of the apps
	VERSION = "version"
)

var settings map[string]string

// initSettings initializes the settings table
func initSettings() {
	settings = make(map[string]string)
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.settings (" +
		"key STRING NOT NULL UNIQUE," +
		"value STRING NOT NULL" +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'settings': ", err)
	}

	rows, err := db.Query("SELECT * FROM gowncloud.settings")
	if err != nil {
		log.Error("Failed to get settings from the database")
		return
	}
	if rows == nil {
		log.Error("Could not load settings")
		return
	}
	defer rows.Close()
	rowCount := 0
	for rows.Next() {
		var key, value string
		err = rows.Scan(&key, &value)
		if err != nil {
			log.Error("Error while reading settings")
			return
		}
		settings[key] = value
		rowCount++
	}
	if rowCount == 0 {
		log.Warn("No settings found")
		makeDefaultSettings()
		return
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the settings rows")
	}

	log.Debug("Initialized 'settings' table")
}

// GetSetting returns the value for key key from the database
func GetSetting(key string) string {
	return settings[key]
}

// UpdateSetting stores the updated setting in the database table
func UpdateSetting(key, value string) error {
	log.Debugf("Update key %v to value %v", key, value)
	if settings[key] == "" {
		log.Error("Trying to update unexisting key")
		return ErrDB
	}
	settings[key] = value
	result, err := db.Exec("UPDATE gowncloud.settings SET value = $1 WHERE key = $2", value, key)
	if err != nil {
		log.Errorf("Error updating key %v", key)
		return ErrDB
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("Error while updating key")
	}
	if rowsAffected != 1 {
		log.Error("Failed to update key")
		return ErrDB
	}
	return nil
}

// makeDefaultSettings generates the default settings and stores them in the database.
func makeDefaultSettings() {
	log.Warn("Generating default settings")

	settings[DEFAULT_ALLOWED_SPACE] = "0"
	settings[DAV_ROOT] = "gowncloud-data"
	settings[VERSION] = ""

	for key, value := range settings {
		_, err := db.Exec("INSERT INTO gowncloud.settings (key, value) VALUES ($1, $2)",
			key, value)
		if err != nil {
			log.Error("Error while storing the settings in the database")
		}
	}
	log.Debug("Initialized 'settings' table with default values")
}
