package core

import (
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
)

// types

const earthCircumferenceMeters = 1000 * 40075.071
const pi = 3.14159265358979323846

type ClientID uuid.UUID
type TokenID uuid.UUID

// Token is a per-session value given to a connected client
type token struct {
	id         TokenID
	timeoutSec int
}

//func (t *token) renew() {
//	t.timeout = time.Now().Unix() + tokenTimeout
//}

// CreateToken creates a new token with timeout
func createToken(timeoutSec int) (*token, error) {
	tok := new(token)
	uid, err := uuid.NewRandom()
	if err != nil {
		return tok, err
	}

	tok.id = TokenID(uid)
	tok.timeoutSec = timeoutSec
	return tok, nil
}

func earthMetersToRadians(m float64) float64 {
	return (2 * pi) * (m / earthCircumferenceMeters)
}

func toHeight(m float64) float64 {
	radiusRadians := earthMetersToRadians(m)
	return (radiusRadians * radiusRadians) / 2
}

// Config struct maintains a topology's configuration
type Config struct {
	// Topology (S2) settings
	searchRadiusMeters float64
	topologyLevel      int
	height             float64

	// Redis settings
	redisUrl        string
	tokenTimeoutSec int
}

// MakeConfig creates a new config object.
func MakeConfig(searchRadiusMeters float64, topologyLevel int) Config {
	c := Config{
		searchRadiusMeters,
		topologyLevel,
		toHeight(searchRadiusMeters),
		"redis://localhost",
		30}
	return c
}

// Topology is the structure that contains all the cells and cell
// entities.
type Topology struct {

	// The configuration for this topology
	config Config

	// Publisher
	publisher Publisher

	// A Cell based subscriber
	cellSubscriber Subscriber

	// A group based subscribers
	groupSubscriber Subscriber

	// Map of connections and the channels they are sunscribed to
	//connections map[Connection]map[string]bool

	// Map of channels and the connections subscribed to
	//channels map[string]map[Connection]bool

	// A connection to the Redis database - used for both storage and publishing data
	//pubconn redis.Conn

	// A connection to the Redis database to received data from subscribed channels
	//subconn redis.PubSubConn

	// A Read/Write mutex for synchronising data between threads
	//lock sync.RWMutex
}

// MakeTopology creates a new topology object using the given
// configuration object.
func MakeTopology(c Config) Topology {
	t := Topology{
		config:          c,
		publisher:       MakePublisher(c.redisUrl),
		cellSubscriber:  MakeSubscriber(c.redisUrl),
		groupSubscriber: MakeSubscriber(c.redisUrl),
		//connections: make(map[Connection]map[string]bool),
		//channels:    make(map[string]map[Connection]bool),
	}
	return t
}

//
// Make a Cell for operation within this topology
//
func (t *Topology) MakeCell() *Cell {
	return MakeCell(&t.config)
}

//
// Connect this topology to the redis backend
//
func (t *Topology) Connect() error {

	err := t.publisher.Connect()
	if err != nil {
		log.Fatal("Failed to connect Publisher to Redis Server(1): ", err)
		return err
	}
	log.Print("Publisher connected to Redis Server URL: " + t.publisher.redisUrl)

	err = t.cellSubscriber.Connect()
	if err != nil {
		log.Fatal("Failed to connect Cell Subscriber to Redis Server(1): ", err)
		return err
	}
	log.Print("Cell Subscriber connected to Redis Server URL: " + t.cellSubscriber.redisUrl)

	err = t.groupSubscriber.Connect()
	if err != nil {
		log.Fatal("Failed to connect Group Subscriber to Redis Server(1): ", err)
		return err
	}
	log.Print("Group Subscriber connected to Redis Server URL: " + t.groupSubscriber.redisUrl)
	return nil
}

//
// Start listening for updates from the topology
//
func (t *Topology) Run() error {

	t.cellSubscriber.Run()
	t.groupSubscriber.Run()

	// TODO: wait for the subscribers to be actually running

	return nil
}

//
// CreateToken creates a new Token for this Topology
// Returns the TokenID created or an Error on failure
//
func (t *Topology) CreateToken(clientId ClientID) (TokenID, error) {

	token, err := createToken(t.config.tokenTimeoutSec)
	if err != nil {
		return token.id, err
	}

	err = t.publisher.SetToken(token.id, clientId, t.config.tokenTimeoutSec)
	if err != nil {
		return token.id, err
	}

	return token.id, nil
}

func (t *Topology) GetClientID(tokenId TokenID) (ClientID, error) {

	return t.publisher.GetClientID(tokenId)
}

//
// Subscribe the Connection 'conn' to the cell 'cell' within this Topology.
// Return an error on failure or nil otherwise.
// This will call Unsubscribe for previous subscribed cells.
//
func (t *Topology) SubscribeToCell(conn Connection, cell *Cell) error {

	// Unsubscribe from any groups first
	_ = t.groupSubscriber.Unsubscribe(conn)

	return t.cellSubscriber.SubscribeToCell(conn, cell)
}

//
// Subscribe the Connection 'conn' to the channel 'groupId' within this Topology.
// Return an error on failure or nil otherwise.
//
func (t *Topology) SubscribeToGroup(conn Connection, groupId string, unsubscribe bool) error {

	// Unsubscribe from any cells first
	_ = t.cellSubscriber.Unsubscribe(conn)

	return t.groupSubscriber.Subscribe(conn, groupId, unsubscribe)
}

//
// Unsubscribe the given connection from all groups
//
func (t *Topology) UnsubscribeFromGroups(conn Connection) error {

	return t.groupSubscriber.Unsubscribe(conn)
}

//
// Unsubscribe the Connection 'conn' from all groups and cells.
//
func (t *Topology) Unsubscribe(conn Connection) error {

	_ = t.cellSubscriber.Unsubscribe(conn)
	_ = t.groupSubscriber.Unsubscribe(conn)
	return nil
}

//
// Broadcast location and message for Entity 'entity' to the topology
//
func (t *Topology) Broadcast(entity *Entity, message []byte) error {

	// Broadcast the message to the cells
	cellIdStr := strconv.FormatUint(uint64(entity.cell.s2cellID), 10)
	err := t.publisher.Publish(cellIdStr, message)
	if err != nil {
		return err
	}

	// Broadcast the message the entities Groups
	for _, group := range entity.groups {
		_ = t.publisher.Publish(group.Uuid, message)
	}

	return nil
}

//
//
//
func (t *Topology) EntityExpired(tokenId TokenID) bool {

	return t.publisher.TokenExpired(tokenId)
}

// Display the given Topology.
func (t *Topology) Display() {
	fmt.Printf("Created Topology {\n  Search Radius: %f meters\n  Topology Level: %d\n  Height: %e meters\n}\n",
		t.config.searchRadiusMeters,
		t.config.topologyLevel,
		t.config.height)
}
