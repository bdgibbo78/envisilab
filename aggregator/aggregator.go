package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

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

	rand.Seed(time.Now().Unix())

	c := core.MakeConfig(250, 15)

	// Create the topology with the given configuration and set
	// the message channel.
	t := core.MakeTopology(c)

	// Create the DataStore object
	dataStore := core.MakeDataStore()
	defer dataStore.Disconnect()
	err := dataStore.Connect()
	if err != nil {
		log.Printf("Failed to connect to datastore: %s", err.Error())
	}

	// Make a context for this aggregator to run within
	ctx := core.MakeDataStoreContext(&t, dataStore)

	err = t.Connect()
	if err != nil {
		log.Print("Failed to connect topology to backend. Exiting...")
		return
	}

	// Run the Topology in a go-routine
	t.Run()

	// Start serving HTTPS version 1 client requests
	core.ServeHTTPS_V1(&ctx)

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
