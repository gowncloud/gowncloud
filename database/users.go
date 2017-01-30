package db

import (
	"database/sql"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

// User represents a user object as stored in the database
type User struct {
	id           int
	username     string
	allowedspace int // allowed storage space for this user in GB
}

// initUsers initializes the users table
func initUsers() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.users (" +
		"id SERIAL UNIQUE, " +
		"username STRING PRIMARY KEY, " +
		"allowedspace INT" +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'users': ", err)
	}

	log.Debug("Initialized 'users' table")
}

// CreateUser creates a new user entry in the database. If the user already exists,
// an error will be returned.
func CreateUser(username string) (*User, error) {
	user := &User{}
	defaultSpace, err := strconv.Atoi(GetSetting(DEFAULT_ALLOWED_SPACE))
	if err != nil {
		log.Error("Could not read default allowed space from settings")
	}
	_, err = db.Exec("INSERT INTO gowncloud.users (username, allowedspace) VALUES ($1, $2)",
		username, defaultSpace)
	if err != nil {
		log.Error("Failed to insert new user in database: ", err)
		return nil, ErrDB
	}

	// retrieve the user from the database to get the ID
	row := db.QueryRow("SELECT * FROM gowncloud.users WHERE username = $1", username)
	err = row.Scan(&user.id, &user.username, &user.allowedspace)
	if err != nil {
		log.Panic("Failed to get user from database: ", err)
		return nil, ErrDB
	}

	return user, nil
}

// GetUser retrieves a user from the database. If the user is not found, a nil value
// will be returned without an error. Callers should check the returned pointer
// before using it and create the user if required.
func GetUser(username string) (*User, error) {
	user := &User{}
	row := db.QueryRow("SELECT * FROM gowncloud.users WHERE username = $1", username)
	err := row.Scan(&user.id, &user.username, &user.allowedspace)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Panic("Failed to get user from database: ", err)
		return nil, ErrDB
	}
	return user, nil
}
