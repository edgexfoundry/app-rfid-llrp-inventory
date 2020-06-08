/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
)

var testTagPro = NewTagProcessor(logger.NewClientStdOut("tag-pro-unit", false, "INFO"))

func TestBasicArrival(t *testing.T) {
	ds := newTestDataset(10, testTagPro)

	ds.readAll(Dev1, 0, rssiWeak, 1)
	ds.updateTagRefs()

	if err := ds.verifyAll(Present, asLocation(Dev1, 0)); err != nil {
		t.Error(err)
	}

	// ensure ALL arrivals WERE generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
}

func TestTagMoveWeakRssi(t *testing.T) {
	ds := newTestDataset(10, testTagPro)
	// start all tags in the back stock
	ds.readAll(Dev1, 0, rssiMin, 1)
	ds.updateTagRefs()
	if err := ds.verifyAll(Present, asLocation(Dev1, 0)); err != nil {
		t.Error(err)
	}
	// ensure arrival events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
	ds.resetEvents()

	// move tags to same facility, different sensor
	ds.readAll(Dev2, 0, rssiStrong, 4)
	if err := ds.verifyAll(Present, asLocation(Dev2, 0)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), MovedType); err != nil {
		t.Error(err)
	}
	ds.resetEvents()

	// test that tag stays at new location even with concurrent reads from weaker sensor
	// MOVE back doesn't happen with weak RSSI
	ds.readAll(Dev3, 1, rssiWeak, 1)
	if err := ds.verifyAll(Present, asLocation(Dev2, 0)); err != nil {
		t.Error(err)
	}
	// ensure no events generated
	if err := ds.verifyNoEvents(); err != nil {
		t.Error(err)
	}
}

func TestMoveAntennaLocation(t *testing.T) {
	antennaIds := []int{1, 4, 33, 15, 99}

	for _, antId := range antennaIds {
		t.Run(fmt.Sprintf("Antenna-%d", antId), func(t *testing.T) {
			ds := newTestDataset(1, testTagPro)

			// start all tags at antenna port 0
			ds.readAll(Dev1, 0, rssiMin, 1)
			ds.updateTagRefs()
			// ensure arrival events generated
			if err := ds.verifyEventPattern(1, ArrivedType); err != nil {
				t.Error(err)
			}
			ds.resetEvents()

			// move tag to a different antenna port on same sensor
			ds.tagReads[0].AntennaId = antId
			ds.readAll(Dev1, antId, rssiStrong, 4)
			expected := asLocation(Dev1, antId)
			if ds.tags[0].Location != expected {
				t.Errorf("tag location was %s, but we expected %s.\n\t%#v",
					ds.tags[0].Location, expected, ds.tags[0])
			}
			// ensure moved events generated
			if err := ds.verifyEventPattern(1, MovedType); err != nil {
				t.Error(err)
			}
			ds.resetEvents()
		})
	}
}

