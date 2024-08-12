package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	//"net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"../core"
	"github.com/google/uuid"
)

var reconnect time.Duration = 10

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func marshalData(data *bytes.Buffer) string {
	// Read how many entities we have
	var num uint32
	binary.Read(data, binary.BigEndian, &num)
	number := int(num)

	response := core.DataResponse{}

	fmt.Printf("Num Activities: %d\n", num)

	var s string
	for i := 0; i < number; i++ {
		activity := core.ActivityFromBuffer(data)
		response.Activities = append(response.Activities, activity)
	}

	js, err := json.MarshalIndent(&response, "", "  ")
	if err != nil {
		fmt.Printf("Error: %e\n", err)
	} else {
		s = string(js)
	}

	return s
}

func main() {
	// Parse arguments
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// A UUID for this client
	clientUUID, _ := uuid.NewRandom()
	serviceUUID, _ := uuid.NewRandom()

	// Create location
	loc := core.MakeLocation(-34.9287, 138.5999, 86.45, 0, time.Now().Unix())

	var tokenId core.TokenID
	var connected = false
	// connect to this socket
	for {
		conn, connectErr := net.Dial("tcp", "127.0.0.1:41111")
		//conn, connectErr := net.Dial("tcp", "128.199.100.216:41111")
		count := 0
		for {
			if connectErr != nil {
				fmt.Printf("Failed to connect: %e\n", connectErr)
				break
			}
			if !connected {
				// We are not connected, so send a SYNC message to the server
				syncReqMsg := core.MakeSyncRequestMsg(clientUUID, serviceUUID, loc)
				bytesSent, err := syncReqMsg.Write(conn)
				if err != nil {
					fmt.Printf("Failed to send SYNC: %e\n", err)
				} else {
					s, _ := json.Marshal(&syncReqMsg)
					fmt.Printf("SYNC REQ: (bytes: %d) Msg=%s\n", bytesSent, string(s))
				}

				// Read a SYNC RES msg
				syncResMsg := core.SyncResponseMsg{}
				err = syncResMsg.Read(conn)
				if err != nil {
					fmt.Printf("Failed to read SYNC RES message: %e.\n", err)
					break
				}

				s, _ := json.Marshal(&syncResMsg)
				fmt.Printf("SYNC RES: %s\n", string(s))

				if syncResMsg.Hdr.UUID == syncReqMsg.Hdr.UUID {
					tokenId = syncResMsg.TokenId
					connected = true
					fmt.Printf("Client Connected. Token=%s\n", uuid.UUID(tokenId).String())
				}
			} else {
				loc.Timestamp = time.Now().Unix()
				dataReqMsg := core.MakeDataRequestMsg(tokenId, loc)
				_, err := dataReqMsg.Write(conn)
				if err != nil {
					fmt.Printf("Failed to send DATA REQ: %e\n", err)
					break
				}

				s, _ := json.Marshal(&dataReqMsg)
				fmt.Printf("DATA REQ: %s\n", string(s))

				// Read a Data Response Msg
				dataResMsg := core.DataResponseMsg{}
				err = dataResMsg.Read(conn)
				if err != nil {
					fmt.Printf("Failed to read DATA RES: %e\n", err)
					break
				}

				sd := marshalData(dataResMsg.Data)
				fmt.Printf("DATA RECEIVED: \n%s\n", sd)
			}

			time.Sleep(1000 * time.Millisecond)
			count++
			if count == 60 && *memprofile != "" {
				count = 0
				break
			}
		}

		connected = false
		fmt.Printf("Reconnecting in %d seconds\n", reconnect)
		time.Sleep(reconnect * time.Second)

		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			defer f.Close()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
		}
	}
}
