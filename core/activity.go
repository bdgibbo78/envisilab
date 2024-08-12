package core

type Activity struct {
	ClientId  ClientID   `json:"id"`
	Locations []Location `json:"locations"`
}

type UserData struct {
	ClientId string   `json:"clientid"`
	Location Location `json:"location"`
}

type UserFilter struct {
	Type  string `json:"filtertype"`
	Value string `json:"filtervalue"`
}

type CellActivity struct {
	Data []UserData
}

func NewActivity() Activity {
	activity := Activity{}
	activity.Locations = make([]Location, 0)
	return activity
}
