/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/sensor"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"
	"sync/atomic"
)

const (
	backStock  = "BackStock"
	salesFloor = "SalesFloor"

	defaultFrequency = 927
)

var (
	rssiMin    = -95 * 10
	rssiMax    = -55 * 10
	rssiStrong = rssiMax - (rssiMax-rssiMin)/3
	rssiWeak   = rssiMin + (rssiMax-rssiMin)/3

	tagSerialCounter uint32
	sensorIdCounter  uint32 = 150000 - 1
)

func generateTestSensor(facilityId string, personality sensor.Personality) *sensor.RSP {
	sensorId := atomic.AddUint32(&sensorIdCounter, 1)

	return &sensor.RSP{
		DeviceID:    fmt.Sprintf("Sensor-%06d", sensorId),
		FacilityID:  facilityId,
		Personality: personality,
	}
}

func generateReadData(lastRead int64) *jsonrpc.TagRead {
	serial := atomic.AddUint32(&tagSerialCounter, 1)

	return &jsonrpc.TagRead{
		Epc:        fmt.Sprintf("EPC%06d", serial),
		Tid:        fmt.Sprintf("TID%06d", serial),
		Frequency:  defaultFrequency,
		Rssi:       rssiMin,
		LastReadOn: lastRead,
	}
}
