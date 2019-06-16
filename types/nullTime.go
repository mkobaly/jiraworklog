package types

// import (
// 	"database/sql/driver"
// 	"time"
// )

// type NullTime struct {
// 	Time  time.Time
// 	Valid bool // Valid is true if Time is not NULL
// }

// // Scan implements the Scanner interface.
// func (nt *NullTime) Scan(value interface{}) error {
// 	nt.Time, nt.Valid = value.(time.Time)
// 	return nil
// }

// // Value implements the driver Valuer interface.
// func (nt NullTime) Value() (driver.Value, error) {
// 	if !nt.Valid {
// 		return nil, nil
// 	}
// 	return nt.Time, nil
// }

// func NewNullTime(t time.Time, valid bool) NullTime {
// 	return NullTime{
// 		Time:  t,
// 		Valid: valid,
// 	}
// }

// // TimeFrom creates a new Time that will always be valid.
// func NullTimeFrom(t time.Time) NullTime {
// 	return NewNullTime(t, true)
// }
