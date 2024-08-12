package main

import (
  "../core"
  "encoding/json"
  "time"

  "github.com/google/uuid"
)


type Peers struct {

    ctx *SimulatorContext
    t *core.Topology
    entities []*core.Entity

}

func MakePeers(ctx *SimulatorContext, cell *core.Cell) Peers {

    peers := Peers{
        ctx: ctx,
        t: ctx.t,
        entities: make([]*core.Entity, 0, 9),
    }

    for _, cid := range cell.GetCellGroup() {
        if cid != cell.GetCellId() {

            // Create a simulated entity for this peer cell
            clientId, _ := uuid.NewRandom()
        	tokenId, _ := uuid.NewRandom()

            peerCell := core.MakeCell(cell.GetConfig())
            peerCell.Update(cid)

            entity := core.MakeEntity(ctx, core.ClientID(clientId), core.TokenID(tokenId), peerCell)
            peers.entities = append(peers.entities, entity)
        }
    }
    return peers
}

func (p *Peers) broadcast(timestamp int64) {

    for _, entity := range p.entities {

        loc := core.CellCenterLocation(entity.GetCell().GetCellId())
        loc.Timestamp = timestamp
        entity.Update(loc)

        clientIdStr := uuid.UUID(entity.GetClientId()).String()

        ud := core.UserData{
            ClientId: clientIdStr,
            Location: loc,
        }
        userMsg, err := json.Marshal(&ud)
		if err == nil {
			p.t.Broadcast(entity, userMsg)
		}
    }
}


type SimulatorContext struct {
    t *core.Topology
    activity core.Activity
    peers map[*core.Entity]Peers
    groups []core.Group
}

func MakeSimulatorContext(t *core.Topology) SimulatorContext {
    ctx := SimulatorContext{
        t: t,
        activity: core.NewActivity(),
        peers: make(map[*core.Entity]Peers),
        groups: make([]core.Group, 0)}

    g1 := core.Group{
        Uuid: "gorilla-racing",
        Name: "Gorillas",
    }
    ctx.groups = append(ctx.groups, g1)
    return ctx
}

func (ctx *SimulatorContext) GetTopology() *core.Topology {
	return ctx.t
}

func (ctx *SimulatorContext) CreateToken(clientId core.ClientID) (core.TokenID, error) {

    token, err := ctx.t.CreateToken(clientId)
    if err != nil {
        return token, err
    }

    return token, nil
}

func (ctx *SimulatorContext) CreateEntity(tokenId core.TokenID, userAgent uint8) (*core.Entity, error) {

	clientId, err := ctx.t.GetClientID(tokenId)
	if err != nil {
		return nil, err
	}

	// Create an entity for the client with a cell structure to handle the data
	entity := core.MakeEntity(ctx, clientId, tokenId, ctx.t.MakeCell())

    // Copy in the created groups
    entity.SetGroups(ctx.groups)

	return entity, nil
}

//
// Subscribe the connection 'conn' to the Cell 'cell'
//
func (ctx *SimulatorContext) SubscribeToCell(conn core.Connection, cell *core.Cell) error {

    entity := conn.GetEntity()

    ctx.peers[entity] = MakePeers(ctx, cell)

    return ctx.t.SubscribeToCell(conn, cell)
}

//
// Subscribe the connection 'conn' to the Group with id groupId.
//
func (ctx *SimulatorContext) SubscribeToGroup(conn core.Connection, groupId string) error {

    return ctx.t.SubscribeToGroup(conn, groupId, true)  // unsubscribe first
}

//
// Unsubscribe the given connection from all groups
//
func (ctx *SimulatorContext) UnsubscribeFromGroups(conn core.Connection) error {

    return ctx.t.UnsubscribeFromGroups(conn)
}

//
// Unsubscribe the connection from all its channels/cells
//
func (ctx *SimulatorContext) Unsubscribe(conn core.Connection) error {

    entity := conn.GetEntity()

    _, ok := ctx.peers[entity]
	if ok {
		delete(ctx.peers, entity)
	}

	return ctx.t.Unsubscribe(conn)
}

//
// Standby
//
func (ctx *SimulatorContext) Standby(entity *core.Entity, message []byte) error {

    peers, ok := ctx.peers[entity]
	if ok {
		peers.broadcast(time.Now().Unix())
	}

    return nil
}

//
// Broadcast the location and message to the topology
//
func (ctx *SimulatorContext) Broadcast(entity *core.Entity, message []byte) error {

    err := ctx.t.Broadcast(entity, message)
    if err != nil {
        return err
    }

    peers, ok := ctx.peers[entity]
	if ok {
		peers.broadcast(time.Now().Unix())
	}

    return nil
}

func (ctx *SimulatorContext) GetClientID(tokenId core.TokenID) (core.ClientID, error) {
	return ctx.t.GetClientID(tokenId)
}

func (ctx *SimulatorContext) GetData(tokenId core.TokenID) (core.Activity, error) {

    // TODO:
    return ctx.activity, nil
}
