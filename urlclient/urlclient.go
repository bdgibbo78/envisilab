package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	//"io/ioutil"
	"os"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"../core"
)

var reconnect time.Duration = 10

type SyncRequest struct {
	Uuid string `json:"clientid"`
	Time int64 `json:"time"`
}

type SyncResponse struct {
	TokenId string `json:"tokenid"`
	Groups []core.Group `json:"groups"`
}

func main() {

	// A UUID for this client
	clientUUID, _ := uuid.NewRandom()
	serviceUUID, _ := uuid.NewRandom()

	var tokenId core.TokenID

	// connect to the REST API
	baseURL := "http://localhost:9443/api/v1/entity"
	baseWS := "ws://localhost:9443/api/v1/entity"
	//baseURL := "http://128.199.100.216:9443/api/v1/entity"
	//baseWS := "ws://128.199.100.216:9443/api/v1/entity"
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	for {

		syncMsg := SyncRequest{Uuid: clientUUID.String(), Time: time.Now().Unix()}

		// We are not connected, so send a SYNC message to the server
		postData, lerr := json.Marshal(&syncMsg)
		if lerr != nil {
			log.Fatalln(lerr)
		}
		request, err := http.NewRequest("POST", baseURL + "/sync", bytes.NewBuffer(postData))
		if err != nil {
			log.Fatalln(err)
		}

		// Set the header information
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Version", "1")
		request.Header.Set("User-Agent", "urlclient-binary")
		request.Header.Set("Services", serviceUUID.String())

		// Get the response JSON
		response, err := client.Do(request)
		if err != nil {
			log.Printf("Failed to post SYNC message: %e.\n", err)
			break
		}

		defer response.Body.Close()
		decoder := json.NewDecoder(response.Body)

		var syncRes SyncResponse
		syncErr := decoder.Decode(&syncRes)
		if syncErr != nil {
			log.Fatal("Failed to sync with server: ", syncErr)
			break
		}

		tokenUUID, err := uuid.Parse(syncRes.TokenId)
		tokenId = core.TokenID(tokenUUID)
		if err != nil {
			log.Fatal("Invalid token: ", err)
		}

		for _, group := range(syncRes.Groups) {
			log.Printf("Group(%s): %s", group.Uuid, group.Name)
		}

		userToken := uuid.UUID(tokenId).String()
		log.Printf("Client Connected. Token=%s\n", userToken)

		//
		// Connect to WebSocket
		//
		url := baseWS + "/" + userToken + "/data"

		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Fatal("dial:", err)
		}
		defer c.Close()

		done := make(chan struct{})
		interrupt := make(chan os.Signal, 1)

		go func() {
			defer close(done)
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					log.Println("read:", err)
					return
				}
				log.Printf("recv: %s", message)
			}
		}()

		ticker := time.NewTicker(time.Second * 3)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// Create location
				loc := core.MakeLocation(-34.9287, 138.5999, 86.45, 0, time.Now().Unix())
				payloadJSON, lerr := json.Marshal(&loc)
				if lerr != nil {
					log.Println("Marshal:", lerr)
				}

				request, err := http.NewRequest("POST", baseURL + "/" + userToken + "/beacon", bytes.NewBuffer(payloadJSON))
				if err != nil {
					log.Fatalln(err)
				}

				// Set the header information
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Content-Version", "1")
				request.Header.Set("User-Agent", "urlclient-binary")
				request.Header.Set("Services", serviceUUID.String())

				// Get the response JSON
				response, err := client.Do(request)
				if err != nil {
					log.Printf("Failed to post BEACON message: %e.\n", err)
					break
				}

				defer response.Body.Close()
//				decoder := json.NewDecoder(response.Body)

//				err := c.WriteMessage(websocket.TextMessage, payloadJSON)
//				if err != nil {
//					log.Println("write:", err)
//					return
//				}
			case <-interrupt:
				log.Println("interrupt")

				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Println("write close:", err)
					return
				}
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
		}

		fmt.Printf("Reconnecting in %d seconds\n", reconnect)
		time.Sleep(reconnect * time.Second)
	}
}
