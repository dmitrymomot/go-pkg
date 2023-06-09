package booking

import (
	"time"
)

type (
	BookRoom struct {
		RoomId    string     `json:"room_id,omitempty"`
		GuestName string     `json:"guest_name,omitempty"`
		StartDate *time.Time `json:"start_date,omitempty"`
		EndDate   *time.Time `json:"end_date,omitempty"`
		UnixTime  int64      `json:"unix_time,omitempty"`
	}

	OrderBeer struct {
		RoomId   string `json:"room_id,omitempty"`
		Count    int64  `json:"count,omitempty"`
		UnixTime int64  `json:"unix_time,omitempty"`
	}
)
