package extender

import (
	"encoding/json"
	"gitlab.srv.pv.km/mailpaas/ttconnector-framework/go/notification"
	"strconv"
)

type Extender interface {
	Extend(message *json.RawMessage, uid int64) (*json.RawMessage, error)
}

type extenderImpl struct {
	db  *notification.Db
	acl string
	ip  string
}

func New() (extender Extender, err error) {
	db, err := notification.Connect("ttconnector", "localhost:10053")
	if err != nil {
		return
	}
	extender = &extenderImpl{
		db: db,
	}
	return
}

func (e *extenderImpl) Extend(message *json.RawMessage, uid int64) (extended *json.RawMessage, err error) {
	var d data
	err = json.Unmarshal([]byte(*message), &d)
	if err != nil {
		return
	}

	var id uint64
	shardKey := uint64(uid)
	for i, msg := range d.Messages {
		id, err = strconv.ParseUint(msg.ID, 10, 64)
		if err != nil {
			return
		}
		var result string
		result, err = e.db.NotificationGetData(shardKey, e.acl, e.ip).Execute(id)
		if err != nil {
			return
		}
		if len(result) > 0 {
			d.Messages[i].AddExtendedData(result)
		}
	}

	marshaled, err := json.Marshal(d)
	if err != nil {
		return
	}

	raw := json.RawMessage(marshaled)
	extended = &raw
	return
}
