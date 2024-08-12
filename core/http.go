package core

import (
	"encoding/json"
	"time"
	"sync"
	"log"
	"fmt"
	"errors"
	"net/http"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Some constants for our web connections
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// The maximum amount of time in seconds that we can tolerate
	toleranceSec = 5

	// Cleanup period
	cleanupPeriodSec = 30 * time.Second
)

func inSync(t int64) bool {
    now := time.Now().Unix()
	if t > now + toleranceSec || t < now - toleranceSec {
		// Not within 5 seconds of the server time.
		log.Printf("Client/Server time not synchronised (%d seconds)", t-now)
		return false
	}
	return true
}

// Function to marshal all TokenID's into a JSON string
func (t *TokenID) MarshalJSON() ([]byte, error) {
	uuidStr := uuid.UUID(*t).String()
	return json.Marshal(uuidStr)
}

type DataResponse struct {
  Activities []Activity `json:"activities"`
}

//
//
//
type Endpoint struct {

    // The context in which this endpoint is running
	ctx Context

	// A map of all entities
	entities map[string]*Entity

	// A Read/Write lock for synchronising entities
	lock sync.RWMutex

}

func MakeEndpoint(ctx Context) Endpoint {
	e := Endpoint{
		ctx: ctx,
		entities: make(map[string]*Entity),
	}
	return e
}

//
//
//
type WebSocketConnection struct {

	// The entity representing this connection
	entity *Entity

	// The context on which this web socket connection is operating
	endpoint *Endpoint

    // The websocket connection for this Client
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func MakeWebSocketConnection(entity *Entity, endpoint *Endpoint, conn *websocket.Conn) *WebSocketConnection {
	c := &WebSocketConnection{
		entity: entity,
		endpoint: endpoint,
		conn: conn,
		send: make(chan []byte, 100),
	}
	return c
}

func (c *WebSocketConnection) GetEntity() *Entity {
	return c.entity
}

// Continually read some data from the web socket connection and publish it
func (c *WebSocketConnection) ReadPump() {

	// Start the Subscription for the associated Entity using the current connection
	go c.entity.subscription.Start(c)

	defer func() {
		// Stop the subscription and hence unsubscribe this connection
		c.entity.subscription.Stop()
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Create a UserFilter instance
		userFilter := UserFilter{}

		// Decode the received user filter
		err = json.Unmarshal(message, &userFilter)
		if err != nil {
			log.Printf("error: %v", err)
	    		continue
	    }

		c.entity.subscription.setUserFilter(&userFilter)
	}
}

// Continually receive subscribed data and write it to the web socket connection
func (c *WebSocketConnection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte{'['})
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{','})
				w.Write(<-c.send)
			}
			w.Write([]byte{']'})

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WebSocketConnection) Write(message []byte) {
	select {
	case c.send <- message:
	default:
		// TODO: what to do if we cant enqueue
	}
}


var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


// ServeHTTP_V1 starts handling HTTPS Version 1 requests from clients
// HTTPS requests require headers to have the following:
// Content-Type: Mime type for data (application/json)
// Content-Version: int (messaging version)
// User-Agent: int (client type)
// UUID: 16 byte uuid
func ServeHTTPS_V1(ctx Context) {

	// Create the endpoint to handle the requests
	endpoint := MakeEndpoint(ctx)

    router := mux.NewRouter()

	router.HandleFunc("/api/v1/entity/sync", endpoint.SyncHandler)
	router.HandleFunc("/api/v1/entity/{tokenid}/standby", endpoint.StandbyHandler)
	router.HandleFunc("/api/v1/entity/{tokenid}/beacon", endpoint.BeaconHandler)
	router.HandleFunc("/api/v1/entity/{tokenid}/data", endpoint.DataHandler)
	//router.HandleFunc("/api/v1/entity/{tokenid}/complete", endpoint.CompleteHandler)
	router.HandleFunc("/api/v1/entity/{tokenid}/download", endpoint.DownloadHandler)
	http.Handle("/", router)

    log.Print("HTTPS Service Started on Port 9443\n")

    // Use ListenAndServeTLS() instead of ListenAndServe() which accepts two extra parameters.
    // We need to specify both the certificate file and the key file (which we've named
    // https-server.crt and https-server.key).
    //err := http.ListenAndServeTLS(":8080", "https-server.crt", "https-server.key", nil);
	err := http.ListenAndServe(":9443", nil)
    if err != nil {
        log.Println(err)
    }
}

type SyncRequest struct {

	// The client UUID
	Uuid string `json:"clientid"`

	// The client current time for sync
	Time int64 `json:"time"`
}


type SyncResponse struct {

	// The generated tokenID for this session
	TokenId string `json:"tokenid"`

	// A list of groups of which the client is a member
	Groups []Group `json:"groups"`
}

//
// Create an Entity for this endpoint, add it to the map and return it
//
func (endpoint *Endpoint) CreateEntity(clientId ClientID, userAgent uint8) (*Entity, error) {

	// Create a new token for the entity
	tokenId, ctxErr := endpoint.ctx.CreateToken(clientId)
    if ctxErr != nil {
		return nil, ctxErr
	}

	// Create the entity with the given token
	entity, err := endpoint.ctx.CreateEntity(tokenId, userAgent)
	if err != nil {
		return nil, err
	}

	// Add the entity to the map of known entities for this Endpoint
	endpoint.lock.Lock()
	defer endpoint.lock.Unlock()
	endpoint.entities[uuid.UUID(tokenId).String()] = entity

	return entity, nil
}

//
// Return the Entity associated with the given tokenIdStr or error on failure.
//
func (endpoint *Endpoint) GetEntity(tokenIdStr string) (*Entity, error) {

	endpoint.lock.RLock()
	defer endpoint.lock.RUnlock()
	entity, ok := endpoint.entities[tokenIdStr]
	if !ok {
		err := errors.New("Could not find entity with tokenId: " + tokenIdStr)
		return nil, err
	}

	return entity, nil
}

func (endpoint *Endpoint) Cleanup() {
	endpoint.lock.Lock()
	defer endpoint.lock.Unlock()

	for s, e  := range endpoint.entities {
		if e.expired() {
			delete(endpoint.entities, s)
		}
	}
}

// SyncHandler handles SYNC JSON requests.
// Headers:
//    Content-Type: application/JSON
//    Content-Version: 1
//    User-Agent: string
//    Services: [uuid1, uuid2, ...] (optional)
// Request Body:
// {
//     client {
//         uuid: <UUID>
//         time: <unix_time_epoch_sec>
//     }
// }
// Response Body:
//    {
//      token {
//        uuid: <token_uuid>,
//        timeout: <timeout>
//      }
//    }
func (endpoint *Endpoint) SyncHandler(w http.ResponseWriter, req *http.Request) {

    if req.Method != http.MethodPost {
		// Sync requests must be POST
		return
	}

    ct := req.Header.Get("Content-Type")
	if ct != "application/json" {
		messageError(w, "Sync: Not a valid JSON request (Content-Type)", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(req.Body)

	var postData SyncRequest
	err := decoder.Decode(&postData)
	if err != nil {
		messageError(w, "Sync: Invalid client sync request: " + err.Error(), http.StatusBadRequest)
		return
	}

    if !inSync(postData.Time) {
		messageError(w, "Sync: Client/Server time not synchronised", http.StatusBadRequest)
		return
	}

	//userAgent := req.Header.Get("User-Agent")

	clientUUID, uuidErr := uuid.Parse(postData.Uuid)
	if uuidErr != nil {
		messageError(w, "Sync: Failed to parse clientUUID: " + uuidErr.Error(), http.StatusBadRequest)
		return
	}

	var userAgent uint8 = 0 // todo:
	entity, entityErr := endpoint.CreateEntity(ClientID(clientUUID), userAgent)
	if entityErr != nil {
		messageError(w, "Sync: Failed to create entity: " + entityErr.Error(), http.StatusForbidden)
		return
	}

	// Set the content-type of the response
	w.Header().Set("Content-Type", "application/json")

	var syncRes SyncResponse
	syncRes.TokenId = uuid.UUID(entity.tokenId).String()
	syncRes.Groups = entity.groups

	//tokenStr := uuid.UUID(tokenId).String()

	log.Printf("Token Acquired: ClientID: %s, TokenID: %s", postData.Uuid, syncRes.TokenId)

	s, _ := json.Marshal(&syncRes)
	fmt.Fprintf(w, string(s))
}

//
//
//
func (endpoint *Endpoint) handleUserData(entity *Entity, w http.ResponseWriter, req *http.Request) *UserData {

	if req.Method != http.MethodPost {
		// Sync requests must be POST
		return nil
	}

	ct := req.Header.Get("Content-Type")
	if ct != "application/json" {
		messageError(w, "Beacon: Not a valid JSON request (Content-Type)", http.StatusBadRequest)
		return nil
	}

	decoder := json.NewDecoder(req.Body)

	// Create the UserData
	clientIdStr := uuid.UUID(entity.clientId).String()
	userData := UserData{ClientId: clientIdStr}

	// Decode the received user location
	err := decoder.Decode(&userData.Location)
	if err != nil {
		messageError(w, "Beacon: Invalid beacon location: " + err.Error(), http.StatusBadRequest)
		return nil
	}

	if !inSync(userData.Location.Timestamp) {
		messageError(w, "Beacon: Client/Server time not synchronised", http.StatusBadRequest)
		return nil
	}

	// Update the entity's new location
	entity.Update(userData.Location)

	return &userData
}

//
//
//
func (endpoint *Endpoint) StandbyHandler(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	tokenIdStr := vars["tokenid"]

	// Get the entity associated with the token
	entity, err := endpoint.GetEntity(tokenIdStr)
	if err != nil {
		messageError(w, "Standby: Failed to get entity: " + err.Error(), http.StatusBadRequest)
		return
	}

	// Handle the UserData from the client
	userData := endpoint.handleUserData(entity, w, req)
	if userData == nil {
		return
	}

	log.Printf("Standby Location: " + ToJSONString(&userData.Location))

	// Marshall the user data into a message to broadcast
	userMsg, lerr := json.Marshal(userData)
	if lerr != nil {
		messageError(w, "Beacon: Failure generating user data: " + lerr.Error(), http.StatusBadRequest)
		return
	}

	err = endpoint.ctx.Standby(entity, userMsg)
	if err != nil {
		messageError(w, "Standby error: " + err.Error(), http.StatusBadRequest)
		return
	}
}

//
//
//
func (endpoint *Endpoint) BeaconHandler(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	tokenIdStr := vars["tokenid"]

	// Get the entity associated with the token
	entity, err := endpoint.GetEntity(tokenIdStr)
	if err != nil {
		messageError(w, "Beacon: Failed to get entity: " + err.Error(), http.StatusBadRequest)
		return
	}

	// Handle the UserData from the client
	userData := endpoint.handleUserData(entity, w, req)
	if userData == nil {
		return
	}

	log.Printf("Beacon Location: " + ToJSONString(&userData.Location))

	// Marshall the user data into a message to broadcast
	userMsg, lerr := json.Marshal(userData)
	if lerr != nil {
		messageError(w, "Beacon: Failure generating user data: " + lerr.Error(), http.StatusBadRequest)
		return
	}

	err = endpoint.ctx.Broadcast(entity, userMsg)
	if err != nil {
		messageError(w, "Broadcast error: " + err.Error(), http.StatusBadRequest)
		return
	}
}

//
//
//
func (endpoint *Endpoint) DataHandler(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	tokenIdStr := vars["tokenid"]

	// Get the entity associated with this token
	entity, err := endpoint.GetEntity(tokenIdStr)
	if err != nil {
		messageError(w, "Data: Failed to acquire entity: " + err.Error(), http.StatusBadRequest)
		return
	}

	// Upgrade the socket to a WebSocket connection
    conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		messageError(w, "Data: Failed to upgrade to websocket: " + err.Error(), http.StatusBadRequest)
		return
	}

	// Make the WebSocket connection and tie it to the new entity
	c := MakeWebSocketConnection(entity, endpoint, conn)

	// Asynchronously, read from and write to the websocket
	go c.ReadPump()
	go c.WritePump()

	log.Printf("Opened data channel for entity with tokenID: %s", tokenIdStr)
}

func (endpoint *Endpoint) DownloadHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		// Complete requests must be Gets
		return
	}

	vars := mux.Vars(req)
	tokenIdStr := vars["tokenid"]

	// Set the content-type of the response
	w.Header().Set("Content-Type", "application/json")
	tokenUUID, uuidErr := uuid.Parse(tokenIdStr)
	if uuidErr != nil {
		messageError(w, uuidErr.Error(), http.StatusBadRequest)
	}

	tokenId := TokenID(tokenUUID)
	activity, err := endpoint.ctx.GetData(tokenId)
	if err != nil {
		messageError(w, err.Error(), http.StatusNotAcceptable)
		return
	}
	js, err := json.Marshal(&activity)
	if err != nil {
		messageError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(js))
}

func (endpoint *Endpoint) Cleaner() {
	ticker := time.NewTicker(cleanupPeriodSec)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			// Remove expired entities
		}
	}
}

func messageError(w http.ResponseWriter, err string, code int) {
	log.Printf("Error: %s\n", err)
	w.WriteHeader(code)
	fmt.Fprintf(w, "{ error: %s }", err)
}
