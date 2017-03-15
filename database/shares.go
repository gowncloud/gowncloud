package db

import (
	"database/sql"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
)

// Share represents a share of a node to a target. The share contains Permissions
// given to the target. The node owner is not maintained in the member share table,
// he/she should be found by getting the owner from the nodes table with the user
// of the node id.
type Share struct {
	ShareID     int
	NodeID      int
	Target      string
	Time        time.Time
	Permissions int
	ShareType   int
}

const (
	USERSHARE = iota
	GROUPSHARE
	_ // Placeholder
	LINKSHARE
)

// initShares initializes the member shares table
func initShares() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.shares (" +
		"shareid SERIAL UNIQUE PRIMARY KEY, " +
		"nodeid INTEGER REFERENCES gowncloud.nodes, " +
		"target STRING NOT NULL, " +
		"time TIMESTAMPTZ NOT NULL," +
		"permissions INTEGER NOT NULL," +
		"sharetype INTEGER NOT NULL" +
		")")
	if err != nil {
		log.Fatal("Failed to create table 'shares': ", err)
	}

	log.Debug("Initialized 'shares' table")
}

// GetShare gets share info from the database for the given share id.
func GetShareById(shareId int) (*Share, error) {
	share := &Share{}
	row := db.QueryRow("SELECT * FROM gowncloud.shares WHERE shareid = $1", shareId)
	err := row.Scan(&share.ShareID, &share.NodeID, &share.Target, &share.Time, &share.Permissions, &share.ShareType)
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

// GetNodeShareToTarget get the share for a node to a target. In case the target is a group,
// shares to subgroups will not be included.
func GetNodeShareToTarget(nodeId int, target string) (*Share, error) {
	share := &Share{}
	row := db.QueryRow("SELECT * FROM gowncloud.shares WHERE nodeid = $1 AND target = $2", nodeId, target)
	err := row.Scan(&share.ShareID, &share.NodeID, &share.Target, &share.Time, &share.Permissions, &share.ShareType)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("Share not found in database for nodeId %v to user %v", nodeId, target)
			return nil, nil
		}
		log.Error("Error getting share from database: ", err)
		return nil, ErrDB
	}
	return share, nil
}

// GetSharesByNodePath returns all shares for a node with the given path
func GetSharesByNodePath(path string) ([]*Share, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.shares WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path = $1)", path)
	if err != nil {
		log.Error("Failed to get Nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readSharesRows(rows)
}

// GetSharesByNodeId gets all the shares for the node id
func GetSharesByNodeId(nodeId int) ([]*Share, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.shares WHERE nodeid = $1", nodeId)
	if err != nil {
		log.Error("Failed to get Nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readSharesRows(rows)
}

// GetSharesToTarget gets all the shares where user is the target
func GetSharesToTarget(target string) ([]*Share, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.shares WHERE target = $1", target)
	if err != nil {
		log.Error("Failed to get Nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readSharesRows(rows)
}

// GetSharesToGroup loads all shares to a group. It also includes subgroups.
func GetSharesToGroup(target string) ([]*Share, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.shares WHERE target LIKE $1 || '%'", target)
	if err != nil {
		log.Error("Failed to get shared nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readSharesRows(rows)
}

// GetAllSharesToUser retuns all the shares to a user, and including the shares
// to the groups where he is a member
func GetAllSharesToUser(username string, groups []string) ([]*Share, error) {
	personalShares, err := GetSharesToTarget(username)
	if err != nil {
		return nil, err
	}
	groupShares := make([]*Share, 0)
	var shares []*Share
	for _, group := range groups {
		shares, err = GetSharesToGroup(group)
		if err != nil {
			return nil, err
		}
		groupShares = append(groupShares, shares...)
	}
	for _, share := range groupShares {
		alreadyFound := false
		for _, uniqueShare := range personalShares {
			if uniqueShare.NodeID == share.NodeID {
				alreadyFound = true
				break
			}
		}
		if !alreadyFound {
			personalShares = append(personalShares, share)
		}
	}
	return personalShares, err
}

// CreateShareToUser creates a new share on the node to the target with permissions.
// Share time is the current system time
func CreateShareToUser(nodeId int, permissions int, target string) (*Share, error) {
	return CreateShare(nodeId, permissions, target, USERSHARE)
}

// CreateShareToGroup creates a new share on the node to the target with permissions.
// Share time is the current system time
func CreateShareToGroup(nodeId int, permissions int, target string) (*Share, error) {
	return CreateShare(nodeId, permissions, target, GROUPSHARE)
}

// DeleteNodeShareToUserFromNodeId deletes the share of the node with nodeId to
// the target.
func DeleteNodeShareToUserFromNodeId(nodeId int, target string) error {
	_, err := db.Exec("DELETE FROM gowncloud.shares WHERE target = $1 AND "+
		"nodeid = $2", target, nodeId)
	if err != nil {
		log.Error("Error while deleting share: ", err)
		return ErrDB
	}
	return nil
}

// DeleteShare removes the share with shareId from the database. It does not remove
// the acutal node.
func DeleteShare(shareId int) error {
	log.Debug("TRY TO DELETE SHARE WITH SHAREID: ", shareId)
	_, err := db.Exec("DELETE FROM gowncloud.shares WHERE shareid = $1", shareId)
	if err != nil {
		log.Error("Error while deleting share: ", err)
		return ErrDB
	}
	return nil
}

// DeleteShareWithPartialId removes the share with shareId ending in partialId,
// if the targeted node is owned by username. It does not remove the actual node
func DeleteShareWithPartialId(partialId int, username string) error {
	strLen := len(strconv.Itoa(partialId))
	log.Debugf("Try to delete share with partial shareId (%v), length of shareId is %v", partialId, strLen)
	_, err := db.Exec("DELETE FROM gowncloud.shares WHERE shareid % POWER(10, $3) = $1 AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.nodes WHERE owner = $2)", partialId, username, strLen)
	if err != nil {
		log.Error("Error while deleting share: ", err)
		return ErrDB
	}
	return nil
}

// GetSharedNodesForUser returns share info on all the nodes of a user that are
// currently being shared
func GetSharedNodesForUser(username string) ([]*Share, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.shares WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE owner = $1)", username)
	if err != nil {
		log.Error("Failed to get shared nodes from the database: ", err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading shares")
		return nil, ErrDB
	}
	defer rows.Close()
	return readSharesRows(rows)
}

// CreateShare creates a new share
func CreateShare(nodeId int, permissions int, target string, sharetype int) (*Share, error) {
	_, err := db.Exec("INSERT INTO gowncloud.shares (nodeid, target, time, permissions, sharetype) "+
		"VALUES ($1, $2, $3, $4, $5)", nodeId, target, time.Now(), permissions, sharetype)
	if err != nil {
		log.Error("Error while creating share: ", err)
		return nil, ErrDB
	}
	return GetNodeShareToTarget(nodeId, target)
}

// readSharesRows reads from *sql.Rows and creates a share info for every rows
func readSharesRows(rows *sql.Rows) ([]*Share, error) {
	shares := make([]*Share, 0)
	for rows.Next() {
		share := &Share{}
		err := rows.Scan(&share.ShareID, &share.NodeID, &share.Target, &share.Time, &share.Permissions, &share.ShareType)
		if err != nil {
			log.Error("Error while reading shares: ", err)
			return nil, ErrDB
		}
		shares = append(shares, share)
	}
	err := rows.Err()
	if err != nil {
		log.Error("Error while reading the shares rows: ", err)
		return nil, err
	}
	return shares, nil
}
