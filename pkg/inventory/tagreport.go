package inventory

import (
	"encoding/hex"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/helper"
	"math"
	"time"
)

const (
	UnknownAntenna = math.MaxInt16
	UnknownRSSI    = -120
)

type AccessReport struct {
	*llrp.ROAccessReport
	DeviceName   string
	OriginMillis int64
	TagReports   []*TagReport
}

func NewAccessReport(deviceName string, origin int64, data *llrp.ROAccessReport) *AccessReport {
	a := AccessReport{
		ROAccessReport: data,
		DeviceName:     deviceName,
		OriginMillis:   origin / int64(time.Millisecond),
		TagReports:     make([]*TagReport, 0, len(data.TagReportData)),
	}
	for _, r := range data.TagReportData {
		a.TagReports = append(a.TagReports, NewTagReport(&r))
	}
	return &a
}

type TagReport struct {
	Data *llrp.TagReportData

	EPC      string
	LastRead int64
	Antenna  int
	RSSI     float64
}

func NewTagReport(r *llrp.TagReportData) *TagReport {
	t := TagReport{
		Data: r,
	}

	if r.EPC96.EPC != nil && len(r.EPC96.EPC) > 0 {
		t.EPC = hex.EncodeToString(r.EPC96.EPC)
	} else {
		t.EPC = hex.EncodeToString(r.EPCData.EPC)
	}

	if r.AntennaID != nil {
		t.Antenna = int(*r.AntennaID)
	} else {
		// todo: document edge-case
		t.Antenna = UnknownAntenna
	}

	if r.LastSeenUTC != nil {
		// Note: LastSeenUTC is in Microseconds and needs to be in Milliseconds
		t.LastRead = int64(uint64(*r.LastSeenUTC) / uint64(1000))
	} else {
		// todo: document edge-case
		t.LastRead = helper.UnixMilliNow()
	}

	if r.PeakRSSI != nil {
		t.RSSI = float64(*r.PeakRSSI)
	} else {
		// todo: document edge-case
		t.RSSI = UnknownRSSI
	}

	return &t
}
