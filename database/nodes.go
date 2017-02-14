package db

import (
	"database/sql"

	log "github.com/Sirupsen/logrus"
)

// Node represents a file or directory, stored on disk by gowncloud.
type Node struct {
	ID       int
	Owner    string
	Path     string
	Isdir    bool
	MimeType string
	Deleted  bool
}

// initNodes initializes the nodes table
func initNodes() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.nodes (" +
		"nodeid SERIAL UNIQUE PRIMARY KEY, " +
		"owner STRING REFERENCES gowncloud.users, " + //all nodes should have an owner
		"path STRING NOT NULL UNIQUE, " +
		"isdir BOOL NOT NULL," +
		"mimetype STRING NOT NULL, " +
		"deleted BOOL NOT NULL " +
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
	err := row.Scan(&node.ID, &node.Owner, &node.Path, &node.Isdir, &node.MimeType, &node.Deleted)
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
func SaveNode(path, owner string, isdir bool, mimetype string) (*Node, error) {
	_, err := db.Exec("INSERT INTO gowncloud.nodes (owner, path, isdir, mimetype, deleted) "+
		"VALUES ($1, $2, $3, $4, false)", owner, path, isdir, mimetype)
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
	// Delete the shares on the node if there are any
	_, err := db.Exec("DELETE FROM gowncloud.membershares WHERE nodeid in ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path = $1)", path)
	if err != nil {
		log.Error("Failed to delete share on node: ", err)
		return ErrDB
	}

	// Delete the trash REFERENCES
	_, err = db.Exec("DELETE FROM gowncloud.trashnodes WHERE nodeid in ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path LIKE $1 || '%')", path)
	if err != nil {
		log.Error("Failed to delete trash reference on node: ", err)
	}

	_, err = db.Exec("DELETE FROM gowncloud.nodes WHERE path LIKE $1 || '%'", path)
	if err != nil {
		log.Error("Failed to delete node: ", err)
		return ErrDB
	}
	return nil
}

// NodeExists checks if a node for the given path exists in the database. returns
// true a node is found, false otherwise. If an error occurs, false is returned
// together with an error
func NodeExists(path string) (bool, error) {
	row := db.QueryRow("SELECT EXISTS (SELECT 1 FROM gowncloud.nodes WHERE path = $1)", path)
	var exists bool
	err := row.Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Error("Failed to get node from database")
		return false, ErrDB
	}
	return exists, nil
}

// GetSharedNode gets the node for the share object
func GetSharedNode(shareId int) (*Node, error) {
	row := db.QueryRow("SELECT * FROM gowncloud.nodes WHERE nodeid in ("+
		"SELECT nodeid FROM gowncloud.membershares WHERE shareid = $1)", shareId)
	node := &Node{}
	err := row.Scan(&node.ID, &node.Owner, &node.Path, &node.Isdir, &node.MimeType, &node.Deleted)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("Node not found in database for shareId ", shareId)
			return nil, nil
		}
		log.Error("Error getting node from database: ", err)
		return nil, ErrDB
	}
	return node, nil
}

// GetSharedNamedNodesToUser finds all nodes where the path ends in nodeName that are
// shared to the sharee
func GetSharedNamedNodesToUser(nodeName, sharee string) ([]*Node, error) {
	nodes := make([]*Node, 0)
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE path LIKE '%' || $1 AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.membershares WHERE sharee = $2)", nodeName, sharee)
	if err != nil {
		log.Error("Failed to get Nodes from the database")
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	for rows.Next() {
		node := &Node{}
		err = rows.Scan(&node.ID, &node.Owner, &node.Path, &node.Isdir, &node.MimeType, &node.Deleted)
		if err != nil {
			log.Error("Error while reading shares")
			return nil, ErrDB
		}
		nodes = append(nodes, node)
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the shares rows")
		return nil, err
	}
	return nodes, nil
}

// MoveNode updates the nodes path in the database.
func MoveNode(originalPath string, targetPath string) error {
	result, err := db.Exec("UPDATE gowncloud.nodes SET path = $1 WHERE path = $2", targetPath, originalPath)
	if err != nil {
		log.Errorf("Error updating path %v: %v", originalPath, err)
		return ErrDB
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("Error while updating path")
	}
	if rowsAffected != 1 {
		log.Error("Failed to update path")
		return ErrDB
	}

	return nil
}
