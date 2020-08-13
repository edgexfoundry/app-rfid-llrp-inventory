/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"encoding/hex"
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"math"
	"sync/atomic"
)

var (
	rssiMin    = float64(-95)
	rssiMax    = float64(-55)
	rssiStrong = rssiMax - math.Floor((rssiMax-rssiMin)/3)
	rssiWeak   = rssiMin + math.Floor((rssiMax-rssiMin)/3)

	tagSerialCounter uint32
	sensorIdCounter  uint32 = 0
)

func generateTestSensor() string {
	sensorID := atomic.AddUint32(&sensorIdCounter, 1)
	return fmt.Sprintf("Sensor-%02X", sensorID)
}

func generateReadData(lastRead int64) *TagReport {
	serial := atomic.AddUint32(&tagSerialCounter, 1)

	// note: ensure even string length
	epcBytes, err := hex.DecodeString(fmt.Sprintf("%024X", serial))
	if err != nil {
		panic(err)
	}

	rssi := llrp.PeakRSSI(rssiMin)
	seen := llrp.LastSeenUTC(lastRead)

	// note: the antenna and device name are always overridden when readTag is called
	return NewTagReport("test-device", &llrp.TagReportData{
		EPC96: llrp.EPC96{
			EPC: epcBytes,
		},
		PeakRSSI:    &rssi,
		LastSeenUTC: &seen,
	})
}
