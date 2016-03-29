package extender

import (
	"encoding/json"
	"gitlab.srv.pv.km/mailpaas/ttconnector-framework/go/notification"
	"strconv"
)

type Extender interface {
	Extend(message []byte, uid int64) ([]byte, error)
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

func (e *extenderImpl) Extend(message []byte, uid int64) (extended []byte, err error) {
	var d data
	err = json.Unmarshal(message, &d)
	if err != nil {
		return
	}

	var id uint64
	shardKey := uint64(uid)
	for _, msg := range d.Messages {
		id, err = strconv.ParseUint(msg.ID, 10, 64)
		if err != nil {
			return
		}
		var result string
		result, err = e.db.NotificationGetData(shardKey, e.acl, e.ip).Execute(id)
		if err != nil {
			return
		}
		d.AddExtendedData(result)
	}

	extended, err = json.Marshal(d)
	return
}
