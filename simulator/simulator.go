package main

import (
	"flag"
	"math/rand"
	"time"
	"log"
	"../core"
)

func main() {

	// Parse arguments
	flag.Parse()

	rand.Seed(time.Now().Unix())

	c := core.MakeConfig(250, 15)

  	// Create the topology with the given configuration and set
	// the message channel.
	t := core.MakeTopology(c) //, messages)

	ctx := MakeSimulatorContext(&t)

	// Connect the topology to the backend
	err := t.Connect()
	if err != nil {
		log.Print("Failed to connect topology to backend. Exiting...")
		return
	}

	// Run the Topology
	t.Run()

	// Start serving HTTPS version 1 client requests
	core.ServeHTTPS_V1(&ctx)
}
