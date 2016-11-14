package libcentrifugo

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shilkin/centrifugo/libcentrifugo/extender"
	"github.com/shilkin/centrifugo/libcentrifugo/logger"
	"github.com/tarantool/go-tarantool"
	"strings"
)

func (p *TarantoolPool) get() (conn *tarantool.Connection, err error) {
	if len(p.pool) == 0 {
		return nil, errors.New("Empty tarantool pool")
	}
	conn = p.pool[p.current]
	p.current++
	p.current = (p.current) % len(p.pool)
	return
}

type TarantoolEngine struct {
	app      *Application
	pool     *TarantoolPool
	extender extender.Extender
	endpoint string
}

type TarantoolEngineConfig struct {
	PoolConfig  TarantoolPoolConfig
	Endpoint    string
	TTConnector extender.Config
}

type TarantoolPool struct {
	pool    []*tarantool.Connection
	config  TarantoolPoolConfig
	current int
}

type TarantoolPoolConfig struct {
	Address  string
	PoolSize int
	Opts     tarantool.Opts
}

/* MessageType
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
	Body   Message `json:"body"`
	Error  string  `json:"error"`
	Method string  `json:"method"`
}

type ServiceMessage struct {
	Action string
	Data   []string
}

type IDs []string

func NewTarantoolEngine(app *Application, conf TarantoolEngineConfig) *TarantoolEngine {
	logger.INFO.Printf("Initializing tarantool connection pool...")
	pool, err := newTarantoolPool(conf.PoolConfig)
	if err != nil {
		logger.FATAL.Fatalln(err)
	}

	extender, err := extender.New(conf.TTConnector)
	if err != nil {
		logger.FATAL.Fatalln(err)
	}

	e := &TarantoolEngine{
		app:      app,
		pool:     pool,
		extender: extender,
		endpoint: conf.Endpoint,
	}

	return e
}

func newTarantoolPool(config TarantoolPoolConfig) (p *TarantoolPool, err error) {
	if config.PoolSize == 0 {
		err = errors.New("Size of tarantool pool is zero")
		return
	}

	p = &TarantoolPool{
		pool:   make([]*tarantool.Connection, config.PoolSize),
		config: config,
	}

	for i := 0; i < config.PoolSize; i++ {
		logger.INFO.Printf("[%d] Connecting to tarantool on %s... [%d]", i, config.Address, config.Opts.MaxReconnects)
		p.pool[i], err = tarantool.Connect(config.Address, config.Opts)
		if err != nil && config.Opts.Reconnect > 0 {
			logger.ERROR.Printf("[%d] connection to tarantool on %s failed with '%s'", i, config.Address, err)
			err = nil // just log and reset error: reconnection inside tarantool.Connect
		}
		if err == nil {
			logger.INFO.Printf("[%d] Connected to tarantool on %s", i, config.Address)
		}
	}

	return
}

// getName returns a name of concrete engine implementation
func (e *TarantoolEngine) name() string {
	return "Tarantool"
}

// publish allows to send message into channel
func (e *TarantoolEngine) publish(chID ChannelID, message []byte) error {
	// Process service messages
	if chID != e.app.config.ControlChannel && chID != e.app.config.AdminChannel {
		newMessage, err := e.processMessage(chID, message)
		if err != nil {
			return err // if no need further processing
		}
		message = newMessage
	}
	// All other messages
	return e.app.handleMsg(chID, message)
}

// subscribe on channel
func (e *TarantoolEngine) subscribe(chID ChannelID) (err error) {
	logger.INFO.Printf("subscribe %s", chID)
	endpoint, err := e.makeEndpointFromChannelID(chID)
	if err != nil {
		logger.ERROR.Printf("subscribe make endpoint string error: %v\n", err.Error())
		return
	}

	conn, err := e.pool.get()
	if err != nil {
		logger.ERROR.Printf("subscribe tarantool pool error: %v\n", err.Error())
		return
	}

	_, err = conn.Call("notification_channel_subscribe", []interface{}{chID, endpoint})

	return
}

// unsubscribe from channel
func (e *TarantoolEngine) unsubscribe(chID ChannelID) (err error) {
	endpoint, err := e.makeEndpointFromChannelID(chID)
	if err != nil {
		logger.ERROR.Printf("unsubscribe make endpoint string error: %v\n", err.Error())
		return
	}

	conn, err := e.pool.get()
	if err != nil {
		logger.ERROR.Printf("unsubscribe tarantool pool error: %v\n", err.Error())
		return
	}

	_, err = conn.Call("notification_channel_unsubscribe", []interface{}{chID, endpoint})

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
	conn, err := e.pool.get()
	if err != nil {
		logger.ERROR.Printf("history tarantool pool error: %v\n", err.Error())
		return nil, err
	}

	history, err := conn.Call("notification_notification_history", []interface{}{chID})
	if err != nil {
		logger.ERROR.Printf("history error: %v\n", err.Error())
		return nil, err
	}

	return processHistory(history)
}

// helpers

type tarantoolHistoryItem struct {
	Count  interface{} `json:"count"`
	Status string      `json:"status"`
	ID     string      `json:"id"`
}

func processHistory(history *tarantool.Response) (msgs []Message, err error) {
	if len(history.Data) == 0 {
		return // history is empty
	}

	data := history.Data[0].([]interface{})
	if len(data) != 2 {
		return // history is empty
	}

	count := data[0]                       // ring counter
	buffer := data[1].(string)             // string buffer
	ring := strings.Split(buffer[1:], ",") // array of IDs

	if len(ring) == 0 {
		return // history buffer is empty [useless?]
	}

	for _, id := range ring {
		encoded, err := json.Marshal(tarantoolHistoryItem{
			Count:  count, // redundancy in each item to pass number of unread notifications
			Status: string(id[0]),
			ID:     string(id[1:]),
		})
		if err != nil {
			logger.ERROR.Println(err)
			continue
		}
		rawMessage := json.RawMessage([]byte(encoded))
		msgs = append(msgs, Message{Data: &rawMessage})
	}

	return
}

func (e *TarantoolEngine) extendMessage(chID ChannelID, message []byte) (newMessage []byte, err error) {
	logger.DEBUG.Printf("try to extend message chID = %s, message = %s", chID, string(message))

	//uid, _, _, err := parseChannelID(chID)
	//if err != nil {
	//	return
	//}

	var m MessageType
	err = json.Unmarshal(message, &m)
	if err != nil {
		return
	}

	extended, err := e.extender.Extend(m.Body.Data, string(chID))
	if extended != nil {
		m.Body.Data = extended
		newMessage, err = json.Marshal(&m)
		logger.DEBUG.Printf("data extended to: %s", string(*m.Body.Data))
	}

	return
}

func (e *TarantoolEngine) processMessage(chID ChannelID, message []byte) (newMessage []byte, err error) {
	newMessage = message // by default, but may be changed

	var msg MessageType
	err = json.Unmarshal(message, &msg)
	if err != nil {
		return
	}

	var srv ServiceMessage
	err = json.Unmarshal(*msg.Body.Data, &srv)
	if err != nil {
		return
	}

	newMessage, err = e.extendMessage(chID, message)
	if err != nil {
		logger.ERROR.Printf("extend message failed with '%s'", err)
	}

	return
}

func (e *TarantoolEngine) makeEndpointFromChannelID(chID ChannelID) (endpoint string, err error) {
	// split chID <centrifugo>.<project>.[$]<uid>_<ringno>
	str := string(chID)
	logger.INFO.Printf("makeEndpointFromChannelID %s", str)
	result := strings.Split(str, ".")
	if len(result) != 3 {
		err = fmt.Errorf("unexpected ChannelID %s", str)
		return
	}
	endpoint = e.endpoint + "/api/" + result[1]
	return
}
