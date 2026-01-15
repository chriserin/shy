package summary

import (
	"time"

	"github.com/chris/shy/pkg/models"
)

// TimePeriod represents a time bucket (Morning, Afternoon, Evening, Night)
type TimePeriod string

const (
	Night      TimePeriod = "Night"
	Morning    TimePeriod = "Morning"
	Afternoon  TimePeriod = "Afternoon"
	Evening    TimePeriod = "Evening"
)

// TimeBucket represents commands in a time period with time range
type TimeBucket struct {
	Period     TimePeriod
	Commands   []models.Command
	FirstTime  int64 // Unix timestamp of first command
	LastTime   int64 // Unix timestamp of last command
}

// GetTimePeriod determines which time period a timestamp falls into
// Uses local timezone for hour calculation
func GetTimePeriod(timestamp int64) TimePeriod {
	t := time.Unix(timestamp, 0)
	hour := t.Hour()

	switch {
	case hour >= 6 && hour < 12:
		return Morning
	case hour >= 12 && hour < 18:
		return Afternoon
	case hour >= 18 && hour < 24:
		return Evening
	default: // 0-5
		return Night
	}
}

// BucketByTimePeriod groups commands by time period (Morning, Afternoon, Evening, Night)
// Returns a map of time period to TimeBucket with commands and time range
func BucketByTimePeriod(commands []models.Command) map[TimePeriod]*TimeBucket {
	buckets := make(map[TimePeriod]*TimeBucket)

	for _, cmd := range commands {
		period := GetTimePeriod(cmd.Timestamp)

		// Initialize bucket if it doesn't exist
		if buckets[period] == nil {
			buckets[period] = &TimeBucket{
				Period:    period,
				Commands:  []models.Command{},
				FirstTime: cmd.Timestamp,
				LastTime:  cmd.Timestamp,
			}
		}

		// Append command
		buckets[period].Commands = append(buckets[period].Commands, cmd)

		// Update time range
		if cmd.Timestamp < buckets[period].FirstTime {
			buckets[period].FirstTime = cmd.Timestamp
		}
		if cmd.Timestamp > buckets[period].LastTime {
			buckets[period].LastTime = cmd.Timestamp
		}
	}

	return buckets
}

// GetOrderedPeriods returns time periods in chronological order
func GetOrderedPeriods() []TimePeriod {
	return []TimePeriod{Night, Morning, Afternoon, Evening}
}
