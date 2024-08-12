package core

import (
  "database/sql"
  "fmt"
  "log"
  "time"

  _ "github.com/lib/pq"
  "github.com/google/uuid"
)

const (
  host = "localhost"
  port = 5432
  user = "data_producer"
  password = "envisilabdataproducer"
  dbname = "envisilab"
)

type DataStore struct {
  dbType string
  dbInfo string
  db *sql.DB
}

// Make a data store and return it
func MakeDataStore() DataStore {
  psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
    host, port, user, password, dbname)

  store := DataStore{dbType: "postgres", dbInfo: psqlInfo, db: nil}
  log.Printf("Datastore Created (type=postgres, host=%s, port=%d)\n", host, port)
  return store
}

func (ds *DataStore) Connect() error {
  db, err := sql.Open("postgres", ds.dbInfo)
  if err != nil {
    return err
  }

  err = db.Ping()
  if err != nil {
    return err
  }
  log.Print("DataStore connected.\n")
  return nil
}

func (ds *DataStore) Disconnect() {
  ds.db.Close()
  log.Print("DataStore disconnected.\n")
}

func (ds *DataStore) IsConnected() bool {
  return ds.db != nil
}

// NewEntity inserts a new entity into the datastore along with its initial location
func (ds *DataStore) NewEntity(clientUUID uuid.UUID, tokenUUID uuid.UUID, userAgent string) error {
  sql := `
INSERT INTO v1.entity (client_uuid, token_uuid, type, created)
VALUES ($1, $2, $3, $4)`
  ds.db.QueryRow(sql, clientUUID, tokenUUID, userAgent, time.Now())

  return nil
}

func (ds *DataStore) LocationData(tokenUUID uuid.UUID, loc Location) error {
  // TODO: It will be a better idea at some stage to bulk import location data.
  sql := `
INSERT INTO v1.location_data (token_uuid, lat, lng, alt, timestamp)
VALUES ($1, $2, $3, $4, $5)`
  stmt, err := ds.db.Prepare(sql)
  if err != nil {
    return err
  }
  defer stmt.Close()

  _, err = stmt.Exec(tokenUUID, loc.Lat, loc.Lng, loc.Alt, time.Unix(loc.Timestamp, 0))
  if err != nil {
    return err
  }
  return nil
}

func (ds *DataStore) GetData(tokenUUID uuid.UUID) ([]Location, error) {
  locations := make([]Location, 0)
  rows, err := ds.db.Query(
    `SELECT lat, lng, alt, timestamp FROM v1.location_data WHERE token_uuid = $1`,
    tokenUUID)
  if err != nil {
    return locations, err
  }

  defer rows.Close()
  for rows.Next() {
    loc := Location{}
    ts := time.Time{}
    err = rows.Scan(&loc.Lat, &loc.Lng, &loc.Alt, &ts)
    if err != nil {
      return locations, err
    }
    loc.Timestamp = ts.Unix()
    locations = append(locations, loc)
  }
  err = rows.Err()
  if err != nil {
    return locations, err
  }
  return locations, nil
}
