package db

import (
	"database/sql"

	log "github.com/Sirupsen/logrus"
)

type TrashNode struct {
	NodeId int
	Owner  string
	Path   string
	IsDir  bool
}

func initTrashNodes() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.trashnodes (" +
		"nodeid INTEGER REFERENCES gowncloud.nodes, " +
		"owner STRING REFERENCES gowncloud.users, " +
		"path STRING NOT NULL UNIQUE, " +
		"isdir BOOL NOT NULL" +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'trashnodes': ", err)
	}

	log.Debug("Initialized 'trashnodes' table")
}

// CreateTrashNode creates a new trash node that links to the original node
func CreateTrashNode(nodeId int, owner string, path string, isDir bool) (*TrashNode, error) {
	_, err := db.Exec("INSERT INTO gowncloud.trashnodes (nodeid, owner, path, isdir) "+
		"VALUES ($1, $2, $3, $4)", nodeId, owner, path, isDir)

	if err != nil {
		log.Error("Error while saving trashnode: ", err)
		return nil, ErrDB
	}

	return GetTrashNode(path)
}

// GetTrashNode gets the trash node at the given path. If no node is found,
// nil is returned without error.
func GetTrashNode(path string) (*TrashNode, error) {
	tn := &TrashNode{}
	row := db.QueryRow("SELECT * FROM gowncloud.trashnodes WHERE nodeid in ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path = $1)", path)
	err := row.Scan(&tn.NodeId, &tn.Owner, &tn.Path, &tn.IsDir)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("No trash node found for path: ", path)
			return nil, nil
		}
		log.Error("Error getting trash node: ", err)
		return nil, ErrDB
	}
	return tn, nil
}

func DeleteTrashNode(path string) error {
	_, err := db.Exec("DELETE FROM gowncloud.trashnodes WHERE path = $1", path)
	if err != nil {
		log.Error("Error while deleting trashnode: ", err)
		return ErrDB
	}
	return nil
}
