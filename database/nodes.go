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
	_, err := db.Exec("DELETE FROM gowncloud.shares WHERE nodeid in ("+
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

	// Delete favorites references
	_, err = db.Exec("DELETE FROM gowncloud.favorites WHERE nodeid in ("+
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
		"SELECT nodeid FROM gowncloud.shares WHERE shareid = $1)", shareId)
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

// GetSharedNamedNodesToTargets returns all the nodes shared to the targets with the
// given name
func GetSharedNamedNodesToTargets(nodeName string, username string, targets []string) ([]*Node, error) {
	nodes, err := getSharedNamedNodesToUser(nodeName, username)
	if err != nil {
		log.Error("Failed to get shared nodes to user: ", err)
		return nil, ErrDB
	}
	for _, target := range targets {
		nodesForTarget, err := getSharedNamedNodesToGroup(nodeName, target)
		if err != nil {
			return nil, err
		}
		for _, nft := range nodesForTarget {
			alreadyFound := false
			for _, node := range nodes {
				if node.ID == nft.ID {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				nodes = append(nodes, nft)
			}
		}
	}
	return nodes, nil
}

func getSharedNamedNodesToUser(nodeName string, user string) ([]*Node, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE path LIKE '%' || $1 AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.shares WHERE target = $2)", nodeName, user)
	if err != nil {
		log.Error("Failed to get Nodes from the database")
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readNodeRows(rows)
}

func getSharedNamedNodesToGroup(nodeName string, target string) ([]*Node, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE path LIKE '%' || $1 AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.shares WHERE target LIKE $2 || '.' || '%')", nodeName, target)
	if err != nil {
		log.Error("Failed to get Nodes from the database")
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readNodeRows(rows)
}

// GetNodesForUserByName returns all the users nodes ending with the given name
func GetNodesForUserByName(nodeName string, username string) ([]*Node, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE path LIKE $1 || '%' "+
		"|| $2", username, nodeName)
	if err != nil {
		log.Error("Failed to get Nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading nodes")
		return nil, ErrDB
	}
	defer rows.Close()
	return readNodeRows(rows)
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
		return ErrDB
	}
	if rowsAffected != 1 {
		log.Error("Failed to update path")
		return ErrDB
	}

	return nil
}

// TransferNode moves a node to a new path and transfers ownership
func TransferNode(originalPath string, targetPath string, newOwner string) error {
	result, err := db.Exec("UPDATE gowncloud.nodes SET path = $1, owner = $2 WHERE path = $3", targetPath, newOwner, originalPath)
	if err != nil {
		log.Errorf("Error updating path %v and owner: %v", originalPath, err)
		return ErrDB
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("Error while updating path and owner")
		return ErrDB
	}
	if rowsAffected != 1 {
		log.Error("Failed to update path and owner")
		return ErrDB
	}
	return nil
}

// SearchNodesByName looks for all nodes where the path contains the query, where
// the user has access
func SearchNodesByName(nodeName string, user string, targets []string) ([]*Node, error) {
	nodes, err := getNodesByNameForUser(nodeName, user) //= make([]*Node, 0)
	if err != nil {
		return nil, err
	}
	for _, t := range targets {
		foundNodes, err := getNodesByNameForGroup(nodeName, t)
		if err != nil {
			return nil, err
		}
		// Make sure we don't add duplicates
		for _, foundNode := range foundNodes {
			alreadyFound := false
			for _, node := range nodes {
				if foundNode.ID == node.ID {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				nodes = append(nodes, foundNode)
			}
		}
	}
	return nodes, nil
}

func getNodesByNameForUser(nodeName string, user string) ([]*Node, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE path ~ ('.*' || $1 || '[^/]*$') AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.nodes WHERE owner = $2 UNION "+
		"SELECT nodeid FROM gowncloud.nodes WHERE path LIKE ("+
		"SELECT path FROM gowncloud.nodes WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.shares WHERE target = $2)) || '%')", nodeName, user)
	if err != nil {
		log.Error("Failed to get Nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading nodes")
		return nil, ErrDB
	}
	defer rows.Close()
	return readNodeRows(rows)
}

// getNodesByNameForTarget returns all nodes where the path contains the query and the
// target has access
func getNodesByNameForGroup(nodeName string, target string) ([]*Node, error) {
	// '.*' || $1 || '[^/]*$' ==> match any characters before the placeholder, then match
	// anything as long as its not a slash until the end of the path. This makes sure
	// we only get nodes where the query is part of the name and not part of the path,
	// else we would also pull in all the child nodes as well.
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE path ~ ('.*' || $1 || '[^/]*$') AND "+
		"nodeid IN ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path LIKE ("+
		"SELECT path FROM gowncloud.nodes WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.shares WHERE target LIKE $2 || '.' || '%')) || '%')", nodeName, target)
	if err != nil {
		log.Error("Failed to get Nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading nodes")
		return nil, ErrDB
	}
	defer rows.Close()
	return readNodeRows(rows)
}

// readNodeRows reads from *sql.Rows and creates a node for every row
func readNodeRows(rows *sql.Rows) ([]*Node, error) {
	nodes := make([]*Node, 0)
	for rows.Next() {
		node := &Node{}
		err := rows.Scan(&node.ID, &node.Owner, &node.Path, &node.Isdir, &node.MimeType, &node.Deleted)
		if err != nil {
			log.Error("Error while reading nodes")
			return nil, ErrDB
		}
		nodes = append(nodes, node)
	}
	err := rows.Err()
	if err != nil {
		log.Error("Error while reading the nodes rows")
		return nil, err
	}
	return nodes, nil
}
