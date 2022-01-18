package inventory

import (
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
)

// ReportInfo holds both pre-existing as well as computed metadata about an incoming ROAccessReport
type ReportInfo struct {
	DeviceName  string
	OriginNanos int64

	offsetMicros int64
	// referenceTimestamp is the same as OriginNanos, but converted to milliseconds
	referenceTimestamp int64
}

// NewReportInfo creates a new ReportInfo based on an EdgeX Reading value
func NewReportInfo(reading *dtos.BaseReading) ReportInfo {
	return ReportInfo{
		DeviceName:         reading.DeviceName,
		OriginNanos:        reading.Origin,
		referenceTimestamp: reading.Origin / int64(time.Millisecond),
	}
}
