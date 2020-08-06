//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/gomodule/redigo/redis"
)

const (
	unknown                = "UNKNOWN"
	pfxTagGen2      string = "tag-gen2:"
	pfxTagReads            = "tag-read-hist:"
	pfxTagLocations        = "tag-locations:"
	pfxTagEVents           = "tag-events:"
)

// todo: add/fix persistence

type TagProcessor struct {
	log      logger.LoggingClient
	cnxPool  *redis.Pool
	tags     map[string]*Tag
	adjuster rssiAdjuster
	mutex    sync.Mutex
}

func NewTagProcessor(lc logger.LoggingClient) *TagProcessor {
	tagPro := &TagProcessor{
		log:      lc,
		tags:     make(map[string]*Tag),
		adjuster: newRssiAdjuster(),
	}

	// tagPro.initializeDB()
	return tagPro
}

func (tagPro *TagProcessor) GetRawInventory() []StaticTag {
	tagPro.mutex.Lock()
	defer tagPro.mutex.Unlock()

	// convert tag map of pointers into a flat array of non-pointers
	res := make([]StaticTag, 0, len(tagPro.tags))
	for _, tag := range tagPro.tags {
		res = append(res, newStaticTag(tag))
	}
	return res
}

func (tagPro *TagProcessor) ProcessReadData(read *Gen2Read) (e Event) {
	tagPro.mutex.Lock()
	defer tagPro.mutex.Unlock()

	tag, exists := tagPro.tags[read.EPC]
	if !exists {
		tag = NewTag(read.EPC)
		tagPro.tags[read.EPC] = tag
	}

	prev := tag.asPreviousTag()
	tag.update(read, &tagPro.adjuster)

	tagPro.log.Debug("Tag updated.",
		"previous", fmt.Sprintf("%+v", prev),
		"current", fmt.Sprintf("%+v", tag))

	switch prev.state {
	case Unknown:
		tag.setState(Present)
		e = Arrived{
			Epc:       read.EPC,
			Timestamp: read.Timestamp,
			DeviceId:  read.DeviceID,
			Location:  read.AsLocation(),
		}

	case Present:
		if prev.location != "" && prev.location != tag.Location {
			e = Moved{
				Epc:          read.EPC,
				Timestamp:    read.Timestamp,
				PrevLocation: prev.location,
				NextLocation: tag.Location,
			}
		}

	}
	return
}

func (tagPro *TagProcessor) initializeDB() {
	pool := redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 80,
		// Dial is an application supplied function for creating and
		// configuring a connection.
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ":6379")
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
	c := pool.Get()
	defer c.Close()
	s, err := redis.String(c.Do("PING"))
	if err != nil {
		panic(err.Error())
	} else {
		tagPro.cnxPool = &pool
		tagPro.log.Info(fmt.Sprintf("Connected to Redis %s", s))
	}
}

func (tagPro *TagProcessor) GetTagRedis(epc string) (t Tag) {

	if tagPro.cnxPool == nil {
		return
	}

	c := tagPro.cnxPool.Get()
	defer c.Close()
	k := pfxTagGen2 + epc
	v, err := redis.String(c.Do("GET", k))
	if err != nil {
		tagPro.log.Error(err.Error())
		return t
	}
	if err = json.Unmarshal([]byte(v), &t); err != nil {
		tagPro.log.Error(err.Error())
	}
	return t
}
