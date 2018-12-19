package syncer

import (
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/siddontang/go-mysql/mysql"

	"github.com/pingcap/tidb-enterprise-tools/dm/config"
	"github.com/pingcap/tidb-enterprise-tools/pkg/streamer"
	"github.com/pingcap/tidb-enterprise-tools/pkg/utils"
)

func (s *Syncer) setInitActiveRelayLog() error {
	if s.binlogType != LocalBinlog {
		return nil
	}

	var (
		pos        mysql.Position
		activeUUID string
		extractPos bool
		err        error
	)

	indexPath := filepath.Join(s.cfg.RelayDir, utils.UUIDIndexFilename)
	uuids, err := utils.ParseUUIDIndex(indexPath)
	if err != nil {
		return errors.Annotatef(err, "UUID index file path %s", indexPath)
	}
	if len(uuids) == 0 {
		return errors.New("no valid relay sub directory exists")
	}

	checkPos := s.checkpoint.GlobalPoint()
	if checkPos.Compare(minCheckpoint) > 0 {
		// continue from previous checkpoint
		pos = checkPos
		extractPos = true
	} else if s.cfg.Mode == config.ModeIncrement {
		// fresh start for task-mode increment
		pos = mysql.Position{
			Name: s.cfg.Meta.BinLogName,
			Pos:  s.cfg.Meta.BinLogPos,
		}
	} else {
		// start from dumper or loader, get current pos from master
		pos, _, err = utils.GetMasterStatus(s.fromDB.db, s.cfg.Flavor)
		if err != nil {
			return errors.Annotatef(err, "get master status")
		}
	}

	if extractPos {
		activeUUID, _, pos, err = streamer.ExtractPos(pos, uuids)
		if err != nil {
			return errors.Trace(err)
		}
	} else {
		var uuid string
		latestUUID := uuids[len(uuids)-1]
		uuid, err = utils.GetServerUUID(s.fromDB.db, s.cfg.Flavor)
		if err != nil {
			return errors.Annotatef(err, "get server UUID")
		}
		// latest should be the current
		if !strings.HasPrefix(latestUUID, uuid) {
			return errors.Errorf("UUID %s not the latest one in UUIDs %v", uuid, uuids)
		}
		activeUUID = latestUUID
	}

	err = s.readerHub.UpdateActiveRelayLog(s.cfg.Name, activeUUID, pos.Name)
	log.Infof("[syncer] current earliest active relay log %s", s.readerHub.EarliestActiveRelayLog())
	return errors.Trace(err)
}

func (s *Syncer) updateActiveRelayLog(pos mysql.Position) error {
	if s.binlogType != LocalBinlog {
		return nil
	}

	indexPath := filepath.Join(s.cfg.RelayDir, utils.UUIDIndexFilename)
	uuids, err := utils.ParseUUIDIndex(indexPath)
	if err != nil {
		return errors.Annotatef(err, "UUID index file path %s", indexPath)
	}
	if len(uuids) == 0 {
		return errors.New("no valid relay sub directory exists")
	}

	activeUUID, _, pos, err := streamer.ExtractPos(pos, uuids)
	if err != nil {
		return errors.Trace(err)
	}

	err = s.readerHub.UpdateActiveRelayLog(s.cfg.Name, activeUUID, pos.Name)
	log.Infof("[syncer] current earliest active relay log %s", s.readerHub.EarliestActiveRelayLog())
	return errors.Trace(err)
}

func (s *Syncer) removeActiveRelayLog() {
	if s.binlogType != LocalBinlog {
		return
	}

	s.readerHub.RemoveActiveRelayLog(s.cfg.Name)
	log.Infof("[syncer] current earliest active relay log %s", s.readerHub.EarliestActiveRelayLog())
}
