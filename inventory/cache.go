/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const (
	TagCacheFile = "/cache/tags.json"
)

func (tp *TagProcessor) Persist(filename string) error {
	tags := tp.Snapshot()
	bytes, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	tp.cacheMu.Lock()
	defer tp.cacheMu.Unlock()

	if err := ioutil.WriteFile(filename, bytes, 0644); err != nil {
		return err
	}

	tp.lc.Debug(fmt.Sprintf("Persisted %d tags to cache", len(tags)))
	return nil
}

func (tp *TagProcessor) Restore(filename string) error {
	tp.cacheMu.Lock()
	defer tp.cacheMu.Unlock()

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var tags []StaticTag
	if err := json.Unmarshal(bytes, &tags); err != nil {
		return err
	}

	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	for _, t := range tags {
		tp.inventory[t.EPC] = t.asTagPtr()
	}

	tp.lc.Info(fmt.Sprintf("Restored %d tags from cache", len(tags)))
	return nil
}
