//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package logutil

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"os"
)

type LogWrap struct {
	logger.LoggingClient
}

type KeyValue struct {
	Key string
	Val interface{}
}

func (lgr LogWrap) ErrIf(cond bool, msg string, params ...KeyValue) bool {
	if !cond {
		return false
	}

	if len(params) > 0 {
		parts := make([]interface{}, len(params)*2)
		for i := range params {
			parts[i*2] = params[i].Key
			parts[i*2+1] = params[i].Val
		}
		lgr.Error(msg, parts...)
	} else {
		lgr.Error(msg)
	}

	return true
}

func (lgr LogWrap) ExitIf(cond bool, msg string, params ...KeyValue) {
	if lgr.ErrIf(cond, msg, params...) {
		os.Exit(1)
	}
}

func (lgr LogWrap) ExitIfErr(err error, msg string, params ...KeyValue) {
	lgr.ExitIf(err != nil, msg, append(params, KeyValue{"error", err})...)
}
