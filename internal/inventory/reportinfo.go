package inventory

import (
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"time"
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
func NewReportInfo(reading *contract.Reading) ReportInfo {
	return ReportInfo{
		DeviceName:         reading.Device,
		OriginNanos:        reading.Origin,
		referenceTimestamp: reading.Origin / int64(time.Millisecond),
	}
}
