package core

import (
	"errors"
	"log"
	"strconv"
	"github.com/google/uuid"
)

type Context interface {

	// Get the topology for this Context
	GetTopology() *Topology

	// Create a token associated with the client Id within this Context
    CreateToken(clientId ClientID) (TokenID, error)

	// Create an Entity for the client 'clientId' and token 'tokenId'.
	CreateEntity(tokenId TokenID, userAgent uint8) (*Entity, error)

	// Get the client id associated with the given token id.
	// The token is considered expired if no client could be found.
	GetClientID(tokenId TokenID) (ClientID, error)

    // Subscribe the connection 'conn' to the cell 'cell'
	SubscribeToCell(conn Connection, cell *Cell) error

	// Subscribe the connection 'conn' to the group with id 'groupId'
	SubscribeToGroup(conn Connection, groupId string) error

	// Unsunscribe the connection 'conn' from all groups
	UnsubscribeFromGroups(conn Connection) error

    // Unsubscribe the connection from all its groups/cells
	Unsubscribe(conn Connection) error

	// Handle the entity being in standby mode
	Standby(entity *Entity, message []byte) error

    // Broadcast the entity location and message to the topology
	Broadcast(entity *Entity, message []byte) error

	// Get the data associated with tokenId in the form of an activity.
	GetData(tokenId TokenID) (Activity, error)

}

type DataStoreContext struct {
	t *Topology
	store DataStore
}

func MakeDataStoreContext(t *Topology, store DataStore) DataStoreContext {
  ctx := DataStoreContext{t: t, store: store}
  return ctx
}

func (ctx *DataStoreContext) GetTopology() *Topology {
	return ctx.t
}

func (ctx *DataStoreContext) CreateToken(clientId ClientID) (TokenID, error) {

	token, entityErr := ctx.t.CreateToken(clientId)
	if entityErr != nil {
		return token, entityErr
	}

	return token, nil
}

func (ctx *DataStoreContext) CreateEntity(tokenId TokenID, userAgent uint8) (*Entity, error) {

	clientId, err := ctx.t.GetClientID(tokenId)
	if err != nil {
		return nil, err
	}

	if !ctx.store.IsConnected() {
		log.Print("CreateEntity: Database is not connected.")
		//return TokenID(uuid.New()), errors.New("Database is not connected.")
	} else {
    	ua := strconv.Itoa(int(userAgent))
		err := ctx.store.NewEntity(uuid.UUID(clientId), uuid.UUID(tokenId), ua)
		if err != nil {
    		return nil, err
		}
	}

	// Create an entity for the client with a cell structure to handle the data
	entity := MakeEntity(ctx, clientId, tokenId, ctx.t.MakeCell())

	// TODO: get groups from database

	return entity, nil
}

func (ctx *DataStoreContext) GetClientID(tokenId TokenID) (ClientID, error) {
	return ctx.t.GetClientID(tokenId)
}

//
// Subscribe the connection 'conn' to the Cell 'cell'
//
func (ctx *DataStoreContext) SubscribeToCell(conn Connection, cell *Cell) error {
	return ctx.t.SubscribeToCell(conn, cell)
}

//
// Subscribe the connection 'conn' to the group with id 'groupId'
//
func (ctx *DataStoreContext) SubscribeToGroup(conn Connection, groupId string) error {
	return ctx.t.SubscribeToGroup(conn, groupId, true)	// unsubscribe first
}

//
// Unsubscribe the given connection from all groups
//
func (ctx *DataStoreContext) UnsubscribeFromGroups(conn Connection) error {

    return ctx.t.UnsubscribeFromGroups(conn)
}

//
// Unsubscribe the connection from all its cells/groups
//
func (ctx *DataStoreContext) Unsubscribe(conn Connection) error {
	return ctx.t.Unsubscribe(conn)
}

//
// Broadcast the location and message to the topology
//
func (ctx *DataStoreContext) Standby(entity *Entity, message []byte) error {

	// TODO:
	return nil
}

//
// Broadcast the location and message to the topology
//
func (ctx *DataStoreContext) Broadcast(entity *Entity, message []byte) error {

	// Insert the location data into the datastore
	if !ctx.store.IsConnected() {
		log.Print("CreateEntity: Database is not connected.")
		//return TokenID(uuid.New()), errors.New("Database is not connected.")
	} else {
		err := ctx.store.LocationData(uuid.UUID(entity.tokenId), entity.location)
		if err != nil {
			return err
		}
	}

    err := ctx.t.Broadcast(entity, message)

	return err
}

func (ctx *DataStoreContext) GetData(tokenId TokenID) (Activity, error) {

	activity := Activity{}

	// Get the client id associated with this token id
	clientId, cerr := ctx.t.GetClientID(tokenId)
	if cerr != nil {
		return activity, cerr
	}

	if !ctx.store.IsConnected() {
		return activity, errors.New("Database is not connected.")
	}

	activity.ClientId = clientId
	var err error = nil
	activity.Locations, err = ctx.store.GetData(uuid.UUID(tokenId))
	return activity, err
}
