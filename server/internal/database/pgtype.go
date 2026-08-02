package database

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// OptionalText converts an empty string to a SQL NULL text value.
func OptionalText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

// OptionalInt64 converts an optional integer to its pgx nullable representation.
func OptionalInt64(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

// Timestamp converts a time to a valid pgx timestamp with time zone.
func Timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

// TimePtr converts a nullable pgx timestamp to an optional time.
func TimePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
