package db

import (
	"database/sql"
	"time"

	log "github.com/Sirupsen/logrus"
)

// MemberShare represents a share of a node to a sharee. The share contains Permissions
// given to the sharee. The node owner is not maintained in the member share table,
// he/she should be found by getting the owner from the nodes table with the user
// of the node id.
type MemberShare struct {
	ShareID     int
	NodeID      int
	Sharee      string
	Time        time.Time
	Permissions int
}

// initMemberShares initializes the member shares table
func initMemberShares() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.membershares (" +
		"shareid SERIAL UNIQUE PRIMARY KEY, " +
		"nodeid INTEGER REFERENCES gowncloud.nodes, " +
		"sharee STRING REFERENCES gowncloud.users, " +
		"time TIMESTAMPTZ NOT NULL," +
		"permissions INTEGER NOT NULL" +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'membershares': ", err)
	}

	log.Debug("Initialized 'membershares' table")
}

// GetShare gets share info from the database for the given share id.
func GetShareById(shareId int) (*MemberShare, error) {
	share := &MemberShare{}
	row := db.QueryRow("SELECT * FROM gowncloud.membershares WHERE shareid = $1", shareId)
	err := row.Scan(&share.ShareID, &share.NodeID, &share.Sharee, &share.Time, &share.Permissions)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("Share not found in database for share id: ", shareId)
			return nil, nil
		}
		log.Error("Error getting share from database: ", err)
		return nil, ErrDB
	}
	return share, nil
}

// GetNodeShareToUser get the share for a node to a user
func GetNodeShareToUser(nodeId int, sharee string) (*MemberShare, error) {
	share := &MemberShare{}
	row := db.QueryRow("SELECT * FROM gowncloud.membershares WHERE nodeid = $1 AND sharee = $2", nodeId, sharee)
	err := row.Scan(&share.ShareID, &share.NodeID, &share.Sharee, &share.Time, &share.Permissions)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("Share not found in database for nodeId %v to user %v", nodeId, sharee)
			return nil, nil
		}
		log.Error("Error getting share from database: ", err)
		return nil, ErrDB
	}
	return share, nil
}

// GetSharesByNodePath returns all shares for a node with the given path
func GetSharesByNodePath(path string) ([]*MemberShare, error) {
	shares := make([]*MemberShare, 0)
	rows, err := db.Query("SELECT * FROM gowncloud.membershares WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path = $1)", path)
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
		share := &MemberShare{}
		err = rows.Scan(&share.ShareID, &share.NodeID, &share.Sharee, &share.Time, &share.Permissions)
		if err != nil {
			log.Error("Error while reading shares")
			return nil, ErrDB
		}
		shares = append(shares, share)
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the shares rows")
		return nil, err
	}
	return shares, nil
}

// GetSharesByNodeId gets all the shares for the node id
func GetSharesByNodeId(nodeId int) ([]*MemberShare, error) {
	shares := make([]*MemberShare, 0)
	rows, err := db.Query("SELECT * FROM gowncloud.membershares WHERE nodeid = $1", nodeId)
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
		share := &MemberShare{}
		err = rows.Scan(&share.ShareID, &share.NodeID, &share.Sharee, &share.Time, &share.Permissions)
		if err != nil {
			log.Error("Error while reading shares")
			return nil, ErrDB
		}
		shares = append(shares, share)
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the shares rows")
		return nil, err
	}
	return shares, nil
}

// GetSharesToUser gets all the shares where user is the sharee
func GetSharesToUser(user string) ([]*MemberShare, error) {
	shares := make([]*MemberShare, 0)
	rows, err := db.Query("SELECT * FROM gowncloud.membershares WHERE sharee = $1", user)
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
		share := &MemberShare{}
		err = rows.Scan(&share.ShareID, &share.NodeID, &share.Sharee, &share.Time, &share.Permissions)
		if err != nil {
			log.Error("Error while reading shares")
			return nil, ErrDB
		}
		shares = append(shares, share)
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the shares rows")
		return nil, err
	}
	return shares, nil
}

// CreateShare creates a new share on the node to the sharee with permissions.
// Share time is the current system time
func CreateShare(nodeId int, permissions int, sharee string) (*MemberShare, error) {
	_, err := db.Exec("INSERT INTO gowncloud.membershares (nodeid, sharee, time, permissions) "+
		"VALUES ($1, $2, $3, $4)", nodeId, sharee, time.Now(), permissions)
	if err != nil {
		log.Error("Error while creating share: ", err)
		return nil, ErrDB
	}
	return GetNodeShareToUser(nodeId, sharee)
}

// DeleteNodeShareToUserFromNodeId deletes the share of the node with nodeId to
// the sharee.
func DeleteNodeShareToUserFromNodeId(nodeId int, sharee string) error {
	_, err := db.Exec("DELETE FROM gowncloud.membershares WHERE sharee = $1 AND "+
		"nodeid = $2", sharee, nodeId)
	if err != nil {
		log.Error("Error while deleting share: ", err)
		return ErrDB
	}
	return nil
}

// DeleteShare removes the share with shareId from the database. It does not removes
// the acutal node.
func DeleteShare(shareId int) error {
	log.Debug("TRY TO DELETE SHARE WITH SHAREID: ", shareId)
	_, err := db.Exec("DELETE FROM gowncloud.membershares WHERE shareid = $1", shareId)
	if err != nil {
		log.Error("Error while deleting share: ", err)
		return ErrDB
	}
	return nil
}
