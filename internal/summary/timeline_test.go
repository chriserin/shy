package summary

import (
	"testing"
	"time"

	"github.com/chris/shy/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// TestBucketBy_Hourly tests grouping commands by hour
func TestBucketBy_Hourly(t *testing.T) {
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

	buckets := BucketBy(commands, Hourly)

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

// TestGetOrderedBuckets tests getting bucket IDs in chronological order
func TestGetOrderedBuckets(t *testing.T) {
	commands := []models.Command{
		{Timestamp: time.Date(2026, 1, 14, 14, 0, 0, 0, time.Local).Unix()},
		{Timestamp: time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix()},
		{Timestamp: time.Date(2026, 1, 14, 20, 0, 0, 0, time.Local).Unix()},
	}

	buckets := BucketBy(commands, Hourly)
	hours := GetOrderedBuckets(buckets)

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

// TestBucketBy_Hourly_TimeRange tests that first/last times are tracked correctly
func TestBucketBy_Hourly_TimeRange(t *testing.T) {
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

	buckets := BucketBy(commands, Hourly)

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

// TestBucketBy_Daily tests grouping commands by day
func TestBucketBy_Daily(t *testing.T) {
	// Given: commands from three different days
	day1Time1 := time.Date(2026, 1, 14, 8, 30, 0, 0, time.Local).Unix()
	day1Time2 := time.Date(2026, 1, 14, 14, 15, 0, 0, time.Local).Unix()
	day2Time := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local).Unix()
	day3Time := time.Date(2026, 1, 16, 16, 45, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{CommandText: "day1-cmd1", Timestamp: day1Time1},
		{CommandText: "day1-cmd2", Timestamp: day1Time2},
		{CommandText: "day2-cmd", Timestamp: day2Time},
		{CommandText: "day3-cmd", Timestamp: day3Time},
	}

	// When: bucketing by day
	buckets := BucketBy(commands, Daily)

	// Then: should have 3 buckets
	assert.Len(t, buckets, 3)

	// And: bucket IDs should be midnight timestamps in local timezone
	day1Midnight := time.Date(2026, 1, 14, 0, 0, 0, 0, time.Local).Unix()
	day2Midnight := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()
	day3Midnight := time.Date(2026, 1, 16, 0, 0, 0, 0, time.Local).Unix()

	// Verify day 1 bucket
	bucket1 := buckets[int(day1Midnight)]
	require.NotNil(t, bucket1)
	assert.Len(t, bucket1.Commands, 2)
	assert.Equal(t, "day1-cmd1", bucket1.Commands[0].CommandText)
	assert.Equal(t, "day1-cmd2", bucket1.Commands[1].CommandText)

	// Verify day 2 bucket
	bucket2 := buckets[int(day2Midnight)]
	require.NotNil(t, bucket2)
	assert.Len(t, bucket2.Commands, 1)
	assert.Equal(t, "day2-cmd", bucket2.Commands[0].CommandText)

	// Verify day 3 bucket
	bucket3 := buckets[int(day3Midnight)]
	require.NotNil(t, bucket3)
	assert.Len(t, bucket3.Commands, 1)
	assert.Equal(t, "day3-cmd", bucket3.Commands[0].CommandText)
}

// TestBucketBy_Daily_MidnightBoundary tests commands right at midnight boundaries
func TestBucketBy_Daily_MidnightBoundary(t *testing.T) {
	// Given: commands at midnight boundaries
	lastSecondDay1 := time.Date(2026, 1, 14, 23, 59, 59, 0, time.Local).Unix()
	firstSecondDay2 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{CommandText: "day1", Timestamp: lastSecondDay1},
		{CommandText: "day2", Timestamp: firstSecondDay2},
	}

	// When: bucketing by day
	buckets := BucketBy(commands, Daily)

	// Then: should have 2 separate buckets
	assert.Len(t, buckets, 2)

	day1Midnight := time.Date(2026, 1, 14, 0, 0, 0, 0, time.Local).Unix()
	day2Midnight := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()

	// Verify they're in different buckets
	assert.NotNil(t, buckets[int(day1Midnight)])
	assert.Equal(t, "day1", buckets[int(day1Midnight)].Commands[0].CommandText)

	assert.NotNil(t, buckets[int(day2Midnight)])
	assert.Equal(t, "day2", buckets[int(day2Midnight)].Commands[0].CommandText)
}

// TestBucketBy_CommandCounts tests that command execution counts are tracked
func TestBucketBy_CommandCounts(t *testing.T) {
	// Given: repeated commands in same hour
	commands := []models.Command{
		{CommandText: "git status", Timestamp: time.Date(2026, 1, 14, 9, 10, 0, 0, time.Local).Unix()},
		{CommandText: "git status", Timestamp: time.Date(2026, 1, 14, 9, 15, 0, 0, time.Local).Unix()},
		{CommandText: "git status", Timestamp: time.Date(2026, 1, 14, 9, 20, 0, 0, time.Local).Unix()},
		{CommandText: "go test", Timestamp: time.Date(2026, 1, 14, 9, 25, 0, 0, time.Local).Unix()},
		{CommandText: "go test", Timestamp: time.Date(2026, 1, 14, 9, 30, 0, 0, time.Local).Unix()},
	}

	// When: bucketing by hour
	buckets := BucketBy(commands, Hourly)

	// Then: command counts should be tracked
	bucket := buckets[9]
	require.NotNil(t, bucket)
	require.NotNil(t, bucket.CommandCounts)

	assert.Equal(t, int64(3), bucket.CommandCounts["git status"])
	assert.Equal(t, int64(2), bucket.CommandCounts["go test"])
}
