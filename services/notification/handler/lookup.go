package handler

import (
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/notification/event"
	"time"
)

type eventLookup struct {
	C        chan *event.Record
	ticker   *time.Ticker
	tenantID uint64
}

func newEventLookup(interval, tid uint64) *eventLookup {
	return &eventLookup{
		C:        make(chan *event.Record),
		ticker:   time.NewTicker(time.Duration(interval) * time.Second),
		tenantID: tid,
	}
}

func (el *eventLookup) close() {
	if el.ticker != nil {
		el.ticker.Stop()
	}

	close(el.C)
}

func (el *eventLookup) subscribeEvent(p broker.Event) error {
	eventRecord := event.Record{}
	err := json.Unmarshal(p.Message().Body, &eventRecord)
	if err != nil {
		err = errors.Unknown(err)
		logger.Errorf("Could not subscribe event. cause: %+v", err)
		return err
	}

	el.C <- &eventRecord
	return nil
}
