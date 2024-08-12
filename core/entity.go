package core

import (
)


type Entity struct {

	// The client id for this Entity
	clientId ClientID

	// The token id given to the Entity
	tokenId TokenID

	// A pointer to the current cell occupied by this entity
	cell *Cell

	// The current location of the Entity
	location  Location

	// A list of groups of which this entity is a member
	groups []Group

	// The subscription used for this Entity
	subscription *Subscription
}

func MakeEntity(ctx Context, clientId ClientID, tokenId TokenID, c *Cell) *Entity {
	e := new(Entity)
	e.clientId = clientId
	e.tokenId = tokenId
	e.cell = c
	e.groups = make([]Group, 0)
	e.subscription = MakeSubscription(ctx)
	return e
}

func (e *Entity) SetGroups(groups []Group) {
	e.groups = groups
}

func (e *Entity) Update(loc Location) {

	e.location = loc
	if !e.cell.Changed(&e.location) {
		return
	}

	cellCpy := *e.cell	// subscription operates on a cell copy
	e.subscription.setCellFilter(&cellCpy)
}

func (e *Entity) GetClientId() ClientID {
	return e.clientId
}

func (e *Entity) GetCell() *Cell {
	return e.cell
}

func (e *Entity) expired() bool {
	return false
}
