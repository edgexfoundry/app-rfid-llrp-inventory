/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

const (
	ArrivedType string = "Arrived"
	MovedType   string = "Moved"
)

type Event interface {
	OfType() string
}

type Arrived struct {
	Epc       string
	Timestamp int64
	DeviceId  string
	Location  string
}

func (a Arrived) OfType() string {
	return ArrivedType
}

type Moved struct {
	Epc          string
	Timestamp    int64
	PrevLocation string
	NextLocation string
}

func (m Moved) OfType() string {
	return MovedType
}
