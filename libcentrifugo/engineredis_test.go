package libcentrifugo

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/shilkin/centrifugo/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/shilkin/centrifugo/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

type testRedisConn struct {
	redis.Conn
}

const (
	testRedisHost     = "127.0.0.1"
	testRedisPort     = "6379"
	testRedisPassword = ""
	testRedisDB       = "9"
	testRedisURL      = "redis://:@127.0.0.1:6379/9"
	testRedisPoolSize = 5
)

func (t testRedisConn) close() error {
	_, err := t.Conn.Do("SELECT", testRedisDB)
	if err != nil {
		return nil
	}
	_, err = t.Conn.Do("FLUSHDB")
	if err != nil {
		return err
	}
	return t.Conn.Close()
}

// Get connection to Redis, select database and if that database not empty
// then panic to prevent existing data corruption.
func dial() testRedisConn {
	addr := net.JoinHostPort(testRedisHost, testRedisPort)
	c, err := redis.DialTimeout("tcp", addr, 0, 1*time.Second, 1*time.Second)
	if err != nil {
		panic(err)
	}

	_, err = c.Do("SELECT", testRedisDB)
	if err != nil {
		c.Close()
		panic(err)
	}

	n, err := redis.Int(c.Do("DBSIZE"))
	if err != nil {
		c.Close()
		panic(err)
	}

	if n != 0 {
		c.Close()
		panic(errors.New("database is not empty, test can not continue"))
	}

	return testRedisConn{c}
}

func testRedisEngine() *RedisEngine {
	app := testApp()
	e := NewRedisEngine(app, testRedisHost, testRedisPort, testRedisPassword, testRedisDB, testRedisURL, true, testRedisPoolSize)
	app.SetEngine(e)
	return e
}

func TestRedisEngine(t *testing.T) {
	c := dial()
	defer c.close()
	e := testRedisEngine()
	assert.Equal(t, e.name(), "Redis")
	assert.Equal(t, nil, e.publish(ChannelID("channel"), []byte("{}")))
	assert.Equal(t, nil, e.subscribe(ChannelID("channel")))
	assert.Equal(t, nil, e.unsubscribe(ChannelID("channel")))
	assert.Equal(t, nil, e.addPresence(ChannelID("channel"), "uid", ClientInfo{}))
	p, err := e.presence(ChannelID("channel"))
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(p))
	assert.Equal(t, nil, e.addHistory(ChannelID("channel"), Message{}, 1, 1))
	h, err := e.history(ChannelID("channel"))
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(h))
	err = e.removePresence(ChannelID("channel"), "uid")
	assert.Equal(t, nil, err)
	apiKey := e.app.config.ChannelPrefix + "." + "api"
	_, err = c.Conn.Do("LPUSH", apiKey, []byte("{}"))
	assert.Equal(t, nil, err)
	_, err = c.Conn.Do("LPUSH", apiKey, []byte("{\"project\": \"test1\"}"))
	assert.Equal(t, nil, err)
}
