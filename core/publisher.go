package core

import (
	"errors"
	"time"
	"github.com/google/uuid"
	"github.com/gomodule/redigo/redis"
)

func MakePool(addr string) *redis.Pool {
  return &redis.Pool{
    MaxIdle: 3,
    IdleTimeout: 240 * time.Second,
    Dial: func () (redis.Conn, error) { return redis.DialURL(addr) },
  }
}

//
// Publisher
//
type Publisher struct {

	// The Redis URL to publish to
	redisUrl string

	// A pool of redis connections
	pool *redis.Pool

}

//
// MakePublisher creates a new topology object using the given configuration object.
//
func MakePublisher(redisUrl string) Publisher {
	p := Publisher{
		redisUrl: redisUrl,
		pool: MakePool(redisUrl),
	}
	return p
}

//
// Connect this publisher to the redis backend
//
func (self *Publisher) Connect() error {

	// Test a connection is established
	conn := self.pool.Get()
	defer conn.Close()

	_, err := conn.Do("PING")
	return err
}

//
// Set a token for client 'clientId'. Token expires after 'tokenTimeoutSec' seconds
//
func (self *Publisher) SetToken(tokenId TokenID, clientId ClientID, tokenTimeoutSec int) error {

	conn := self.pool.Get()
	defer conn.Close()

    // Set the entity's tokenId/clientId in redis with timeout
	_, err := conn.Do("SETEX", uuid.UUID(tokenId).String(), tokenTimeoutSec, uuid.UUID(clientId).String())

	if err != nil {
		return err
	}

	return nil
}

//
// Get the ClientId associated with the given tokenID
//
func (self *Publisher) GetClientID(tokenId TokenID) (ClientID, error) {

	var cid ClientID

	conn := self.pool.Get()
	defer conn.Close()

	val, err := conn.Do("GET", uuid.UUID(tokenId).String())
	if err != nil {
		return cid, err
	}

	strVal, ok := val.([]byte) 	// type assertion
	if !ok {
		return cid, errors.New("Invalid client id - not a valid type")
	}

	clientUUID, uuidErr := uuid.Parse(string(strVal))
	if uuidErr != nil {
		return cid, uuidErr
	}

	return ClientID(clientUUID), nil
}

//
// Publish a message for Entity 'entity' to the topology on channel 'channel'
//
func (self *Publisher) Publish(channel string, message []byte) error {

	conn := self.pool.Get()
	defer conn.Close()

	_, err := conn.Do("PUBLISH", channel, message)
	return err
}

//
// Return true if the given token has expired. Return false otherwise.
//
func (self *Publisher) TokenExpired(tokenId TokenID) bool {

	conn := self.pool.Get()
	defer conn.Close()

	val, _ := conn.Do("GET", uuid.UUID(tokenId).String())
	return val == nil
}
