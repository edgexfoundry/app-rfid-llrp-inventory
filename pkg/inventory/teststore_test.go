/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"encoding/hex"
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
	"sync/atomic"
)

const (
	backStock  = "BackStock"
	salesFloor = "SalesFloor"
)

var (
	rssiMin    = -95
	rssiMax    = -55
	rssiStrong = rssiMax - (rssiMax-rssiMin)/3
	rssiWeak   = rssiMin + (rssiMax-rssiMin)/3

	tagSerialCounter uint32
	sensorIdCounter  uint32 = 150000 - 1
)

func generateTestSensor(facilityID string, personality sensor.Personality) *sensor.Sensor {
	sensorID := atomic.AddUint32(&sensorIdCounter, 1)

	return &sensor.Sensor{
		DeviceID:   fmt.Sprintf("Sensor-%06d", sensorID),
		FacilityID: facilityID,
		// todo: handle persoanlity
		//Personality: personality,
	}
}

func generateReadData(lastRead int64, antennaID int) *llrp.TagReportData {
	serial := atomic.AddUint32(&tagSerialCounter, 1)

	epcBytes, err := hex.DecodeString(fmt.Sprintf("EPC%06d", serial))
	if err != nil {
		panic(err)
	}

	ant := llrp.AntennaID(antennaID)
	rssi := llrp.PeakRSSI(rssiMin)
	seen := llrp.LastSeenUTC(lastRead)

	return &llrp.TagReportData{
		EPC96: llrp.EPC96{
			EPC: epcBytes,
		},
		AntennaID:   &ant,
		PeakRSSI:    &rssi,
		LastSeenUTC: &seen,
	}
}
