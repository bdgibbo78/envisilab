package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"../core"
)

var reconnect time.Duration = 10

func marshalData(data *bytes.Buffer) string {
	// Read how many locations we have
	var num uint32
	binary.Read(data, binary.BigEndian, &num)
	number := int(num)

	fmt.Printf("Num Locations: %d\n", num)

	var s string
	for i := 0; i < number; i++ {
		loc := core.Location{}
		core.Deserialize(&loc, data)
		js, _ := json.Marshal(&loc)
		s += string(js) + "\n"
	}

	return s
}

func main() {

	for {
		// connect to this socket
		conn, err := net.Dial("tcp", "127.0.0.1:41112")
		//conn, err := net.Dial("tcp", "128.199.100.216:41112")
		for {
			if err != nil {
				fmt.Printf("Failed to connect: %e\n", err)
				break
			}
			// Read a forwarded Data Msg
			dataMsg := core.DataResponseMsg{}
			err = dataMsg.Read(conn)
			if err != nil {
				fmt.Printf("Failed to read DATA msg: %e\n", err)
				break
			}

			sd := marshalData(dataMsg.Data)
			fmt.Printf("DATA RECEIVED: \n%s\n", sd)
		}

		fmt.Printf("Reconnecting in %d seconds\n", reconnect)
		time.Sleep(reconnect * time.Second)
	}
}
