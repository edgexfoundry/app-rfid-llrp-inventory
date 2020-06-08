/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"sync/atomic"
)

const (
	Dev1 = "device01"
	Dev2 = "device02"
	Dev3 = "device03"
)

func asLocation(devId string, antId int) string {
	return devId + ":" + string(antId)
}

var (
	rssiMin    = -95 * 10
	rssiMax    = -55 * 10
	rssiStrong = rssiMax - (rssiMax-rssiMin)/3
	rssiWeak   = rssiMin + (rssiMax-rssiMin)/3

	tagSerialCounter uint32
)

func standardReadData(timestamp int64) *Gen2Read {
	return customReadData(Dev1, 0, rssiMin, timestamp)
}

func customReadData(dev string, ant int, rssi int, timestamp int64) *Gen2Read {
	serial := atomic.AddUint32(&tagSerialCounter, 1)

	return &Gen2Read{
		Epc:       fmt.Sprintf("EPC%06d", serial),
		Tid:       fmt.Sprintf("TID%06d", serial),
		User:      fmt.Sprintf("USR%06d", serial),
		Reserved:  fmt.Sprintf("RES%06d", serial),
		DeviceId:  dev,
		AntennaId: ant,
		Timestamp: timestamp,
		Rssi:      rssi,
	}
}
