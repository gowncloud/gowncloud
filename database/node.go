package db

import (
	"database/sql"

	log "github.com/Sirupsen/logrus"
)

// Node represents a file or directory, stored on disk by gowncloud.
type Node struct {
	ID    int
	Owner string
	Path  string
	Isdir bool
}

// initNodes initializes the nodes table
func initNodes() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.nodes (" +
		"id SERIAL UNIQUE, " +
		"owner STRING NOT NULL, " + //all nodes should have an owner
		"path STRING NOT NULL UNIQUE, " +
		"isdir BOOL NOT NULL" +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'nodes': ", err)
	}

	log.Debug("Initialized 'nodes' table")
}

// GetNode get the node with the given path from the database. If no node is found
// a nil object is returned
func GetNode(path string) (*Node, error) {
	node := &Node{}
	row := db.QueryRow("SELECT * FROM gowncloud.nodes WHERE path = $1", path)
	err := row.Scan(&node.ID, &node.Owner, &node.Path, &node.Isdir)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("Node not found in database for path: ", path)
			return nil, nil
		}
		log.Error("Error getting node from database: ", err)
		return nil, ErrDB
	}
	return node, nil
}

// SaveNode saves a new node in the database
func SaveNode(path, owner string, isdir bool) (*Node, error) {
	_, err := db.Exec("INSERT INTO gowncloud.nodes (owner, path, isdir) VALUES ($1, $2, $3)",
		owner, path, isdir)
	if err != nil {
		log.Error("Error while saving node: ", err)
		return nil, ErrDB
	}

	return GetNode(path)
}

// DeleteNode deletes a node for the given path from the database. DeleteNode
// retuns an error when failing to delete an existing node in the database. If
// no error is returned, the client can be sure no more node with the given path
// is present in the database when this function returns.
func DeleteNode(path string) error {
	_, err := db.Exec("DELETE FROM gowncloud.nodes WHERE path LIKE $1 || '%'", path)
	if err != nil {
		log.Error("Failed to delete directory node and children: ", err)
		return ErrDB
	}
	return nil
}
