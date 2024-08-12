package core

import (
	"sync"
	"strconv"
	"github.com/gomodule/redigo/redis"
)


type Subscription struct {

	// The Context on which this Subscription is operating
	ctx Context

	// The current cell
	cell *Cell

	// Subscription filter.
	filter *UserFilter

	// Input
	input chan interface{}
}

func MakeSubscription(ctx Context) *Subscription {
	s := Subscription{
		ctx: ctx,
		cell: nil,
		filter: &UserFilter{Type: "local", Value: ""},
		input: make(chan interface{}),
	}
	return &s
}

//
// Start this subscription operating on the given connection.
//
func (self *Subscription) Start(conn Connection) {
	for {
		//log.Print("Objected received 1")
		var obj interface{}
		select {
		case obj = <- self.input:
			switch obj.(type) {
			case *Cell:
				cell, ok := obj.(*Cell)	// type assertion
				if ok {
					self.cell = cell	// remember for next time
					if self.filter.Type == "local" {
						// An update to the cell
						self.ctx.SubscribeToCell(conn, cell)
					}
				}
			case *UserFilter:
				uf, ok := obj.(*UserFilter)	// type assertion
				if ok {
					if uf.Type != self.filter.Type || uf.Value != self.filter.Value {
						// Filter has changed
						if uf.Type == "group" {
							// Subscribe to group
							self.ctx.SubscribeToGroup(conn, uf.Value)
						} else if uf.Type == "local" && self.cell != nil {
							// Subscribe to cell
							self.ctx.SubscribeToCell(conn, self.cell)
						}
					}
					self.filter = uf
				}
			case string:
				s, ok := obj.(string) 	// type assertion
				if ok && s == "stop" {
					break
				}
			}
		}
	}

	// Subscription finished, so unsubscribe the connection
	self.ctx.Unsubscribe(conn)
}

func (self *Subscription) Stop() {
	select {
	case self.input <- "stop":
	default:
	}
}

func (self *Subscription) setCellFilter(cell *Cell) {
	select {
	case self.input <- cell:
	default:
	}
}

func (self *Subscription) setUserFilter(filter *UserFilter) {
	select {
	case self.input <- filter:
	default:
	}
}



//
// Subscriber
//
type Subscriber struct {

	// The Redis URL to publish to
	redisUrl string

	// Map of connections and the channels they are sunscribed to
	connections map[Connection]map[string]bool

	// Map of channels and the connections subscribed to
	channels map[string]map[Connection]bool

	// A connection to the Redis database to received data from subscribed channels
	subconn redis.PubSubConn

	// A Read/Write mutex for synchronising data between threads
	lock sync.RWMutex
}

//
// MakeSubscriber creates a new topology object using the given redis url
//
func MakeSubscriber(redisUrl string) Subscriber {
	s := Subscriber{
		redisUrl: redisUrl,
		connections: make(map[Connection]map[string]bool),
		channels:    make(map[string]map[Connection]bool)}
	return s
}

//
// Connect this topology to the redis backend
//
func (self *Subscriber) Connect() error {

	c, err := redis.DialURL(self.redisUrl)
	if err != nil {
		return err
	}
	self.subconn = redis.PubSubConn{c}
	return nil
}

//
// Start listening for published data
//
func (self *Subscriber) Run() error {

	go func() {
		for {
			switch v := self.subconn.Receive().(type) {
			case redis.Message:
				self.lock.RLock()
				for conn := range self.channels[v.Channel] {
					conn.Write(v.Data)
				}
				self.lock.RUnlock()
			case error:
				panic(v)
			}
		}
	}()

	return nil
}

//
// Subscribe the Connection 'conn' to the cell 'cell' within this Topology.
// Return an error on failure or nil otherwise.
// This will call Unsubscribe for previous subscribed cells.
//
func (self *Subscriber) SubscribeToCell(conn Connection, cell *Cell) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	// Unsubscribe from old cells
	err := self.unsubscribeNoLock(conn)
	if err != nil {
		return err
	}

	// Subscribe to the new cell group
	for _, cellId := range cell.cellGroup {
		cellIdStr := strconv.FormatUint(uint64(cellId), 10)
		err = self.subscribeNoLock(conn, cellIdStr)
		if err != nil {
			return err
		}
	}
	return nil
}

//
// Subscribe the Connection 'conn' to the channel 'channel' within this Topology.
// Return an error on failure or nil otherwise.
//
func (self *Subscriber) Subscribe(conn Connection, channel string, unsubscribe bool) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if unsubscribe {
		// Unsubscribe from all other channels
		_ = self.unsubscribeNoLock(conn)
	}

	return self.subscribeNoLock(conn, channel)
}

//
//
//
func (self *Subscriber) subscribeNoLock(conn Connection, channel string) error {

	_, ok := self.connections[conn]
	if !ok {
		self.connections[conn] = map[string]bool{}
	}
	self.connections[conn][channel] = true

	_, ok = self.channels[channel]
	if !ok {
		self.channels[channel] = map[Connection]bool{}
		err := self.subconn.Subscribe(channel)
		if err != nil {
			return err
		}
	}
	self.channels[channel][conn] = true
	//log.Printf("Subscribed to %s", channel)
	return nil
}

//
//
//
func (self *Subscriber) Unsubscribe(conn Connection) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	return self.unsubscribeNoLock(conn)
}

//
//
//
func (self *Subscriber) unsubscribeNoLock(conn Connection) error {

	for c := range self.connections[conn] {
		delete(self.channels[c], conn)
		if len(self.channels[c]) == 0 {
			delete(self.channels, c)
			err := self.subconn.Unsubscribe(c)
			if err != nil {
				return err
			}
		}
	}
	delete(self.connections, conn)
	return nil
}
