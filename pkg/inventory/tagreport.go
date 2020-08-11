package inventory

import (
	"encoding/hex"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
)

type TagReport struct {
	*llrp.TagReportData
	deviceName string
}

func NewTagReport(deviceName string, data *llrp.TagReportData) *TagReport {
	return &TagReport{
		TagReportData: data,
		deviceName:    deviceName,
	}
}

func (r *TagReport) EPC() string {
	if r.EPC96.EPC != nil && len(r.EPC96.EPC) > 0 {
		return hex.EncodeToString(r.EPC96.EPC)
	} else {
		return hex.EncodeToString(r.EPCData.EPC)
	}
}
