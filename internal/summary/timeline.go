package summary

import (
	"fmt"
	"sort"
	"time"

	"github.com/chris/shy/pkg/models"
)

// HourBucket represents commands in an hour with time range
type HourBucket struct {
	Hour       int // 0-23
	Commands   []models.Command
	FirstTime  int64 // Unix timestamp of first command
	LastTime   int64 // Unix timestamp of last command
}

// GetHour extracts the hour (0-23) from a Unix timestamp
// Uses local timezone for hour calculation
func GetHour(timestamp int64) int {
	t := time.Unix(timestamp, 0)
	return t.Hour()
}

// BucketByHour groups commands by hour of the day (0-23)
// Returns a map of hour to HourBucket with commands and time range
func BucketByHour(commands []models.Command) map[int]*HourBucket {
	buckets := make(map[int]*HourBucket)

	for _, cmd := range commands {
		hour := GetHour(cmd.Timestamp)

		// Initialize bucket if it doesn't exist
		if buckets[hour] == nil {
			buckets[hour] = &HourBucket{
				Hour:      hour,
				Commands:  []models.Command{},
				FirstTime: cmd.Timestamp,
				LastTime:  cmd.Timestamp,
			}
		}

		// Append command
		buckets[hour].Commands = append(buckets[hour].Commands, cmd)

		// Update time range
		if cmd.Timestamp < buckets[hour].FirstTime {
			buckets[hour].FirstTime = cmd.Timestamp
		}
		if cmd.Timestamp > buckets[hour].LastTime {
			buckets[hour].LastTime = cmd.Timestamp
		}
	}

	return buckets
}

// GetOrderedHours returns hours that have commands, sorted chronologically
func GetOrderedHours(buckets map[int]*HourBucket) []int {
	hours := make([]int, 0, len(buckets))
	for hour := range buckets {
		hours = append(hours, hour)
	}
	sort.Ints(hours)
	return hours
}

// FormatHour formats an hour as "8am", "2pm", "12pm", "12am"
func FormatHour(hour int) string {
	if hour == 0 {
		return "12am"
	} else if hour < 12 {
		return fmt.Sprintf("%dam", hour)
	} else if hour == 12 {
		return "12pm"
	} else {
		return fmt.Sprintf("%dpm", hour-12)
	}
}

// Legacy functions for backward compatibility with old time period system
// These are kept to avoid breaking existing code that might reference them

// TimePeriod represents a time bucket (Morning, Afternoon, Evening, Night)
type TimePeriod string

const (
	Night     TimePeriod = "Night"
	Morning   TimePeriod = "Morning"
	Afternoon TimePeriod = "Afternoon"
	Evening   TimePeriod = "Evening"
)

// TimeBucket represents commands in a time period with time range
type TimeBucket struct {
	Period    TimePeriod
	Commands  []models.Command
	FirstTime int64 // Unix timestamp of first command
	LastTime  int64 // Unix timestamp of last command
}

// GetTimePeriod determines which time period a timestamp falls into
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

// BucketByTimePeriod groups commands by time period (deprecated, use BucketByHour)
func BucketByTimePeriod(commands []models.Command) map[TimePeriod]*TimeBucket {
	buckets := make(map[TimePeriod]*TimeBucket)

	for _, cmd := range commands {
		period := GetTimePeriod(cmd.Timestamp)

		if buckets[period] == nil {
			buckets[period] = &TimeBucket{
				Period:    period,
				Commands:  []models.Command{},
				FirstTime: cmd.Timestamp,
				LastTime:  cmd.Timestamp,
			}
		}

		buckets[period].Commands = append(buckets[period].Commands, cmd)

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
