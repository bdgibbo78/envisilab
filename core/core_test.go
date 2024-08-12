package core

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestAggregator(t *testing.T) {

	c := MakeConfig(250, 15)
	top := MakeTopology(c)
	//topology.Display()

	numEntities := 1000
	tokens := make([]Token, 0, numEntities)

	// Create entities
	start := time.Now().Unix()
	for i := 0; i < numEntities; i++ {

		loc1 := MakeLocation(-34.9287, 138.5999, 86.45, 0, time.Now().Unix())
		tok, _ := top.CreateEntity(loc1)
		tokens = append(tokens, tok)
	}
	end := time.Now().Unix()
	fmt.Printf("Init Time: %d\n", end-start)

	start = time.Now().Unix()
	for _, tok := range tokens {

		loc2 := MakeLocation(-34.9297, 138.5998, 86.56, 0, time.Now().Unix())
		data := new(bytes.Buffer)
		top.UpdateAndRetrieveData(tok, loc2, data)

		top.DestroyEntity(tok)
	}
	end = time.Now().Unix()
	fmt.Printf("Update Time: %d\n", end-start)
}
