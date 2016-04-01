package extender

import (
	"encoding/json"
	"gitlab.srv.pv.km/mailpaas/ttconnector-framework/go/notification"
)

type Extender interface {
	Extend(message *json.RawMessage, uid int64) (*json.RawMessage, error)
}

type extenderImpl struct {
	db  *notification.Db
	acl string
	ip  string
}

func New(config Config) (extender Extender, err error) {
	err = validate(config)
	if err != nil {
		return
	}

	db, err := notification.Connect(config.ServiceName, config.AggrNode+":"+config.AggrPort)
	if err != nil {
		return
	}

	extender = &extenderImpl{
		db:  db,
		acl: config.ACL,
	}
	return
}

func (e *extenderImpl) Extend(message *json.RawMessage, uid int64) (extended *json.RawMessage, err error) {
	var d data
	err = json.Unmarshal([]byte(*message), &d)
	if err != nil {
		return
	}

	shardKey := uint64(uid)
	for i, msg := range d.Messages {
		var result string
		result, err = e.db.NotificationGetData(shardKey, e.acl, e.ip).Execute(msg.ID)
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
