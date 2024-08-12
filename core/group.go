package core

import (
    //"github.com/google/uuid"
)

type Group struct {

	// The unique id for the group
	Uuid string `json:"groupid"`

	// The name of the group
	Name string `json:"name"`
}
