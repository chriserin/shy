package summary

import (
	"testing"
	"time"

	"github.com/chris/shy/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetTimePeriod_Morning tests morning time period (6am-12pm)
func TestGetTimePeriod_Morning(t *testing.T) {
	// 6:00 AM
	ts := time.Date(2026, 1, 14, 6, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Morning, GetTimePeriod(ts))

	// 11:59 AM
	ts = time.Date(2026, 1, 14, 11, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Morning, GetTimePeriod(ts))

	// 9:30 AM
	ts = time.Date(2026, 1, 14, 9, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, Morning, GetTimePeriod(ts))
}

// TestGetTimePeriod_Afternoon tests afternoon time period (12pm-6pm)
func TestGetTimePeriod_Afternoon(t *testing.T) {
	// 12:00 PM
	ts := time.Date(2026, 1, 14, 12, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Afternoon, GetTimePeriod(ts))

	// 5:59 PM
	ts = time.Date(2026, 1, 14, 17, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Afternoon, GetTimePeriod(ts))

	// 2:30 PM
	ts = time.Date(2026, 1, 14, 14, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, Afternoon, GetTimePeriod(ts))
}

// TestGetTimePeriod_Evening tests evening time period (6pm-12am)
func TestGetTimePeriod_Evening(t *testing.T) {
	// 6:00 PM
	ts := time.Date(2026, 1, 14, 18, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Evening, GetTimePeriod(ts))

	// 11:59 PM
	ts = time.Date(2026, 1, 14, 23, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Evening, GetTimePeriod(ts))

	// 9:30 PM
	ts = time.Date(2026, 1, 14, 21, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, Evening, GetTimePeriod(ts))
}

// TestGetTimePeriod_Night tests night time period (12am-6am)
func TestGetTimePeriod_Night(t *testing.T) {
	// 12:00 AM
	ts := time.Date(2026, 1, 14, 0, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Night, GetTimePeriod(ts))

	// 5:59 AM
	ts = time.Date(2026, 1, 14, 5, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Night, GetTimePeriod(ts))

	// 3:30 AM
	ts = time.Date(2026, 1, 14, 3, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, Night, GetTimePeriod(ts))
}

// TestGetTimePeriod_Boundaries tests exact boundary times
func TestGetTimePeriod_Boundaries(t *testing.T) {
	// 5:59:59 AM - should be Night
	ts := time.Date(2026, 1, 14, 5, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Night, GetTimePeriod(ts))

	// 6:00:00 AM - should be Morning
	ts = time.Date(2026, 1, 14, 6, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Morning, GetTimePeriod(ts))

	// 11:59:59 AM - should be Morning
	ts = time.Date(2026, 1, 14, 11, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Morning, GetTimePeriod(ts))

	// 12:00:00 PM - should be Afternoon
	ts = time.Date(2026, 1, 14, 12, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Afternoon, GetTimePeriod(ts))

	// 17:59:59 PM - should be Afternoon
	ts = time.Date(2026, 1, 14, 17, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Afternoon, GetTimePeriod(ts))

	// 18:00:00 PM - should be Evening
	ts = time.Date(2026, 1, 14, 18, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Evening, GetTimePeriod(ts))

	// 23:59:59 PM - should be Evening
	ts = time.Date(2026, 1, 14, 23, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, Evening, GetTimePeriod(ts))

	// 00:00:00 AM (next day) - should be Night
	ts = time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, Night, GetTimePeriod(ts))
}

// TestBucketByTimePeriod_AllPeriods tests bucketing across all time periods
func TestBucketByTimePeriod_AllPeriods(t *testing.T) {
	// Given: commands distributed across all time periods
	commands := []models.Command{
		{
			CommandText: "git commit",
			Timestamp:   time.Date(2026, 1, 14, 2, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "git status",
			Timestamp:   time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go test",
			Timestamp:   time.Date(2026, 1, 14, 13, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "git push",
			Timestamp:   time.Date(2026, 1, 14, 19, 0, 0, 0, time.Local).Unix(),
		},
	}

	// When: bucketing by time period
	buckets := BucketByTimePeriod(commands)

	// Then: should have 4 buckets
	require.Len(t, buckets, 4)

	// And: Night bucket should have 1 command
	require.NotNil(t, buckets[Night])
	assert.Len(t, buckets[Night].Commands, 1)
	assert.Equal(t, "git commit", buckets[Night].Commands[0].CommandText)

	// And: Morning bucket should have 1 command
	require.NotNil(t, buckets[Morning])
	assert.Len(t, buckets[Morning].Commands, 1)
	assert.Equal(t, "git status", buckets[Morning].Commands[0].CommandText)

	// And: Afternoon bucket should have 1 command
	require.NotNil(t, buckets[Afternoon])
	assert.Len(t, buckets[Afternoon].Commands, 1)
	assert.Equal(t, "go test", buckets[Afternoon].Commands[0].CommandText)

	// And: Evening bucket should have 1 command
	require.NotNil(t, buckets[Evening])
	assert.Len(t, buckets[Evening].Commands, 1)
	assert.Equal(t, "git push", buckets[Evening].Commands[0].CommandText)
}

// TestBucketByTimePeriod_SinglePeriod tests bucketing in one time period
func TestBucketByTimePeriod_SinglePeriod(t *testing.T) {
	// Given: commands all in morning
	commands := []models.Command{
		{
			CommandText: "git status",
			Timestamp:   time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go build",
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go test",
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	// When: bucketing by time period
	buckets := BucketByTimePeriod(commands)

	// Then: should have only 1 bucket
	require.Len(t, buckets, 1)

	// And: Morning bucket should have all 3 commands
	require.NotNil(t, buckets[Morning])
	assert.Len(t, buckets[Morning].Commands, 3)
	assert.Equal(t, "git status", buckets[Morning].Commands[0].CommandText)
	assert.Equal(t, "go build", buckets[Morning].Commands[1].CommandText)
	assert.Equal(t, "go test", buckets[Morning].Commands[2].CommandText)
}

// TestBucketByTimePeriod_TimeRange tests first/last time tracking
func TestBucketByTimePeriod_TimeRange(t *testing.T) {
	// Given: commands with varying timestamps in morning
	firstTime := time.Date(2026, 1, 14, 8, 23, 15, 0, time.Local).Unix()
	middleTime := time.Date(2026, 1, 14, 9, 30, 0, 0, time.Local).Unix()
	lastTime := time.Date(2026, 1, 14, 11, 47, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{
			CommandText: "git status",
			Timestamp:   firstTime,
		},
		{
			CommandText: "go build",
			Timestamp:   middleTime,
		},
		{
			CommandText: "go test",
			Timestamp:   lastTime,
		},
	}

	// When: bucketing by time period
	buckets := BucketByTimePeriod(commands)

	// Then: Morning bucket should track correct time range
	require.NotNil(t, buckets[Morning])
	assert.Equal(t, firstTime, buckets[Morning].FirstTime)
	assert.Equal(t, lastTime, buckets[Morning].LastTime)
}

// TestBucketByTimePeriod_TimeRangeOutOfOrder tests time range with out-of-order commands
func TestBucketByTimePeriod_TimeRangeOutOfOrder(t *testing.T) {
	// Given: commands inserted out of chronological order
	firstTime := time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix()
	middleTime := time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix()
	lastTime := time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{
			CommandText: "middle",
			Timestamp:   middleTime,
		},
		{
			CommandText: "first",
			Timestamp:   firstTime,
		},
		{
			CommandText: "last",
			Timestamp:   lastTime,
		},
	}

	// When: bucketing by time period
	buckets := BucketByTimePeriod(commands)

	// Then: Morning bucket should still track correct first/last
	require.NotNil(t, buckets[Morning])
	assert.Equal(t, firstTime, buckets[Morning].FirstTime)
	assert.Equal(t, lastTime, buckets[Morning].LastTime)
}

// TestBucketByTimePeriod_EmptyCommands tests empty command list
func TestBucketByTimePeriod_EmptyCommands(t *testing.T) {
	// Given: empty command list
	commands := []models.Command{}

	// When: bucketing by time period
	buckets := BucketByTimePeriod(commands)

	// Then: should have empty buckets map
	assert.Empty(t, buckets)
}

// TestBucketByTimePeriod_SingleCommand tests single command
func TestBucketByTimePeriod_SingleCommand(t *testing.T) {
	// Given: single command
	timestamp := time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix()
	commands := []models.Command{
		{
			CommandText: "go build",
			Timestamp:   timestamp,
		},
	}

	// When: bucketing by time period
	buckets := BucketByTimePeriod(commands)

	// Then: should have one bucket with one command
	require.Len(t, buckets, 1)
	require.NotNil(t, buckets[Morning])
	assert.Len(t, buckets[Morning].Commands, 1)

	// And: first and last time should be the same
	assert.Equal(t, timestamp, buckets[Morning].FirstTime)
	assert.Equal(t, timestamp, buckets[Morning].LastTime)
}

// TestGetOrderedPeriods tests chronological period ordering
func TestGetOrderedPeriods(t *testing.T) {
	// When: getting ordered periods
	periods := GetOrderedPeriods()

	// Then: should be in chronological order
	require.Len(t, periods, 4)
	assert.Equal(t, Night, periods[0])
	assert.Equal(t, Morning, periods[1])
	assert.Equal(t, Afternoon, periods[2])
	assert.Equal(t, Evening, periods[3])
}

// TestGetHour tests extracting hour from timestamps
func TestGetHour(t *testing.T) {
	tests := []struct {
		name string
		hour int
		want int
	}{
		{"midnight", 0, 0},
		{"1am", 1, 1},
		{"noon", 12, 12},
		{"1pm", 13, 13},
		{"11pm", 23, 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Date(2026, 1, 14, tt.hour, 30, 0, 0, time.Local).Unix()
			got := GetHour(ts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBucketByHour tests grouping commands by hour
func TestBucketByHour(t *testing.T) {
	commands := []models.Command{
		{
			CommandText: "cmd1",
			Timestamp:   time.Date(2026, 1, 14, 8, 15, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd2",
			Timestamp:   time.Date(2026, 1, 14, 8, 45, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd3",
			Timestamp:   time.Date(2026, 1, 14, 9, 10, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd4",
			Timestamp:   time.Date(2026, 1, 14, 14, 30, 0, 0, time.Local).Unix(),
		},
	}

	buckets := BucketByHour(commands)

	// Should have 3 buckets (8, 9, 14)
	assert.Len(t, buckets, 3)

	// Hour 8 should have 2 commands
	assert.NotNil(t, buckets[8])
	assert.Equal(t, 2, len(buckets[8].Commands))
	assert.Equal(t, "cmd1", buckets[8].Commands[0].CommandText)
	assert.Equal(t, "cmd2", buckets[8].Commands[1].CommandText)

	// Hour 9 should have 1 command
	assert.NotNil(t, buckets[9])
	assert.Equal(t, 1, len(buckets[9].Commands))
	assert.Equal(t, "cmd3", buckets[9].Commands[0].CommandText)

	// Hour 14 should have 1 command
	assert.NotNil(t, buckets[14])
	assert.Equal(t, 1, len(buckets[14].Commands))
	assert.Equal(t, "cmd4", buckets[14].Commands[0].CommandText)
}

// TestGetOrderedHours tests getting hours in chronological order
func TestGetOrderedHours(t *testing.T) {
	commands := []models.Command{
		{Timestamp: time.Date(2026, 1, 14, 14, 0, 0, 0, time.Local).Unix()},
		{Timestamp: time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix()},
		{Timestamp: time.Date(2026, 1, 14, 20, 0, 0, 0, time.Local).Unix()},
	}

	buckets := BucketByHour(commands)
	hours := GetOrderedHours(buckets)

	// Should be sorted: 8, 14, 20
	require.Len(t, hours, 3)
	assert.Equal(t, 8, hours[0])
	assert.Equal(t, 14, hours[1])
	assert.Equal(t, 20, hours[2])
}

// TestFormatHour tests hour formatting
func TestFormatHour(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, "12am"},
		{1, "1am"},
		{11, "11am"},
		{12, "12pm"},
		{13, "1pm"},
		{23, "11pm"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatHour(tt.hour)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestHourBucketTimeRange tests that first/last times are tracked correctly
func TestHourBucketTimeRange(t *testing.T) {
	commands := []models.Command{
		{
			CommandText: "cmd1",
			Timestamp:   time.Date(2026, 1, 14, 8, 10, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd2",
			Timestamp:   time.Date(2026, 1, 14, 8, 50, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd3",
			Timestamp:   time.Date(2026, 1, 14, 8, 30, 0, 0, time.Local).Unix(),
		},
	}

	buckets := BucketByHour(commands)

	bucket := buckets[8]
	assert.NotNil(t, bucket)

	// First time should be 8:10
	firstTime := time.Unix(bucket.FirstTime, 0)
	assert.Equal(t, 8, firstTime.Hour())
	assert.Equal(t, 10, firstTime.Minute())

	// Last time should be 8:50
	lastTime := time.Unix(bucket.LastTime, 0)
	assert.Equal(t, 8, lastTime.Hour())
	assert.Equal(t, 50, lastTime.Minute())
}
