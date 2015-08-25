package libcentrifugo

import (
	"github.com/shilkin/centrifugo/libcentrifugo/logger"
	"github.com/shilkin/go-tarantool"
	"errors"
	"strings"
	"strconv"
	"encoding/json"
)


func (p *TarantoolPool) get() (conn *tarantool.Connection, err error) {
	if p.conn == nil {
		return nil, errors.New("Empty pool")
	}
	return p.conn, nil

	/*if len(p.pool) == 0 {
		return nil, errors.New("Empty tarantool pool")
	}
	conn = p.pool[p.current]
	p.current++
	p.current = (p.current)%len(p.pool)
	return */
}

type TarantoolEngine struct {
	app  *Application
	pool *TarantoolPool
	endpoint string
}

type TarantoolPool struct {
	conn *tarantool.Connection
	config TarantoolPoolConfig
}

type TarantoolEngineConfig struct {
	PoolConfig TarantoolPoolConfig
	Endpoint string	
}

type TarantoolPoolConfig struct {
	Address string
	PoolSize int
	Opts tarantool.Opts
}

/*
		{
			"body": {
				"uid":"026c380d-13e1-47d9-42d2-e2dc0e41e8d5",
				"timestamp":"1440434259",
				"info":{
					"user":"3",
					"client":"83309b33-deb7-48ff-76c6-04b10e6a6523",
					"default_info":null,
					"channel_info": {
						"channel_extra_info_example":"you can add additional JSON data when authorizing"
					}
				},
				"channel":"$3_0",
				"data": {
						"Action":"mark",
						"Data":["00000000000000395684"]
					},
				"client":"83309b33-deb7-48ff-76c6-04b10e6a6523"
			},
			"error":null,
			"method":"message"
		}
*/

type MessageType struct {
	Body BodyType
	Error string
	Method string	
}

type BodyType struct {
	Uid string
	Timestamp string
	Info json.RawMessage

	Channel string
	Data json.RawMessage // !!!
	Client string
}

type ServiceMessage struct {
	Action string
	Data []string
	// Data json.RawMessage
}

type IDs []string

func NewTarantoolEngine(app *Application, conf TarantoolEngineConfig) *TarantoolEngine {
	pool, err := newTarantoolPool(conf.PoolConfig)
	if err != nil {
		logger.FATAL.Fatalln(err)
	}

	e := &TarantoolEngine{
		app: app,
		pool: pool,
		endpoint: conf.Endpoint,
	}

	return e
}

func newTarantoolPool(config TarantoolPoolConfig) (*TarantoolPool, error) {
	var err error
	p := new(TarantoolPool)
	p.config = config
	p.conn, err = tarantool.Connect(config.Address, config.Opts)

	if err != nil {
		logger.ERROR.Printf("tarantool.Connect: %v", err.Error())
	}	

	return p, nil

	/*
	if config.PoolSize == 0 {
		return nil, errors.New("Size of tarantool pool is zero")
	}
 	
 	p = &TarantoolPool{
		pool: make([]*tarantool.Connection, config.PoolSize),
		config: config,
	}

	for i := 0; i < config.PoolSize; i++ {
		p.pool[i], err = tarantool.Connect(config.Address, config.Opts) // tmp ignore error
		if err != nil {
			return
		}
	}
	return p, nil
	*/
}

// getName returns a name of concrete engine implementation
func (e *TarantoolEngine) name() string {
	return "Tarantool"
}

// publish allows to send message into channel
func (e *TarantoolEngine) publish(chID ChannelID, message []byte) error {
	/*
		message: 
			action: mark, push
			params:	[id,...]
	*/

	// Process service messages
	if chID != e.app.config.ControlChannel && chID != e.app.config.AdminChannel {
		var msg MessageType
		err := json.Unmarshal(message, &msg)
		if err == nil {
			var srv ServiceMessage
			err = json.Unmarshal(msg.Body.Data, &srv)
			// logger.DEBUG.Println("len(srv) = ", len(srv))
			// empty := ServiceMessage{}
			if srv.Action != "" && err == nil {
				logger.DEBUG.Printf("publish: %t\n", srv)
				var functionName string

				switch(srv.Action) {
				case "mark": 
					functionName = "notification_mark"
				case "push":
					functionName = "notification_push"
				default:
					return e.app.handleMsg(chID, message)	
				}

				// logger.DEBUG.Printf("publish: %v\n", string(message))
				uid, ringno, err := parseChannelID(chID)
				if err != nil {
					logger.DEBUG.Println("Wow!")
					return err
				}
				
				/*var ids IDs
				err = json.Unmarshal(srv.Data, &ids)
				if err != nil {
					return err
				}
				logger.DEBUG.Printf("%s: %v\n", functionName, ids)
				*/

				// TODO: further processing
				conn, err := e.pool.get()
				if err != nil {
					return err
				}

				for _, id := range(srv.Data) {
					_, err = conn.Call(functionName, []interface{}{uid, ringno, id})
					if err != nil {
						logger.ERROR.Printf("%s\n", err.Error())
					}
				}
				return nil
			}
		}
		// goto OTHER	
	}
	// OTHER:
	// All other messages
	return e.app.handleMsg(chID, message)
}

// subscribe on channel
func (e *TarantoolEngine) subscribe(chID ChannelID) (err error) {
	logger.DEBUG.Printf("subscribe: %v\n", chID)

	uid, ringno, err := parseChannelID(chID)
	if err != nil {
		return
	}

	conn, err := e.pool.get()
	if err != nil {
		return
	}

	_, err = conn.Call("notification_subscribe",  []interface{}{uid, ringno, e.endpoint});
	
	return
}

// unsubscribe from channel
func (e *TarantoolEngine) unsubscribe(chID ChannelID) (err error) {
	logger.DEBUG.Printf("unsubscribe: %v\n", chID)

	uid, ringno, err := parseChannelID(chID)
	if err != nil {
		return
	}
	
	conn, err := e.pool.get()
	if err != nil {
		return
	}

	_, err = conn.Call("notification_unsubscribe", []interface{}{uid, ringno, e.endpoint});
	
	return	
}

// addPresence sets or updates presence info for connection with uid
func (e *TarantoolEngine) addPresence(chID ChannelID, uid ConnID, info ClientInfo) (err error) {
	// not implemented
	return 
}

// removePresence removes presence information for connection with uid
func (e *TarantoolEngine) removePresence(chID ChannelID, uid ConnID) (err error) {
	// not implemented
	return
}

// getPresence returns actual presence information for channel
func (e *TarantoolEngine) presence(chID ChannelID) (result map[ConnID]ClientInfo, err error) {
	// not implemented
	return
}

// addHistory adds message into channel history and takes care about history size
func (e *TarantoolEngine) addHistory(chID ChannelID, message Message, size, lifetime int64) (err error) {
	// not implemented
	return
}

// getHistory returns history messages for channel
// return empty slice
// all history pushed via publish
func (e *TarantoolEngine) history(chID ChannelID) (msgs []Message, err error) {
	logger.DEBUG.Printf("history: %v\n", chID)

	uid, ringno, err := parseChannelID(chID)
	if err != nil {
		logger.ERROR.Printf("history parse chID error: %v\n", err.Error())
		return
	}

	conn, err := e.pool.get()
	if err != nil {
		logger.ERROR.Printf("history get conn error: %v\n", err.Error())
		return
	}

	_, err = conn.Call("notification_read", []interface{}{uid, ringno, e.endpoint});
	if err != nil {
		logger.ERROR.Printf("history call stored proc error: %v\n", err.Error())
	}
	
	return
}

// helpers
func parseChannelID(chID ChannelID) (uid, ringno int64, err error) {
	// split chID <blahblah>.[$]<uid>:<ringno>
	str := string(chID)

	dotIndex := strings.LastIndex(str, ".")
	if dotIndex >= 0 {
		str = str[dotIndex+1:]
	}

	separator := "_"
	prefix := "$"
	if strings.HasPrefix(str, prefix) {
		str = strings.TrimLeft(str, prefix)
	}
	channel := strings.Split(str, separator)

	uid, err = strconv.ParseInt(channel[0], 10, 64)
	if err != nil {
		return
	}
	ringno, err = strconv.ParseInt(channel[1], 10, 64)
	if err != nil {
		return
	}
	return uid, ringno, nil
}
