package core


type Connection interface {

    // Return the Entity associated with this Connection
    GetEntity() *Entity

    // Continually read some data from the connection and publish it
    ReadPump()

    // Continually receive subscribed data and write it to the connection
    WritePump()

    // Write data to this connection
    Write(message []byte)

}
