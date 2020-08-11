/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package jsonrpc

import (
	"encoding/json"
	"errors"
)

const (
	RpcVersion = "2.0"
)

var (
	//ErrInvalidVersion error returned when JsonRpc version is not 2.0
	ErrInvalidVersion = errors.New("invalid jsonrpc version")
	//ErrMissingMethod error returned when method field is missing or empty
	ErrMissingMethod = errors.New("missing or empty method field")
	//ErrMissingID error returned when id field is missing or empty
	ErrMissingID = errors.New("missing or empty id field")
)

type Message interface {
	Validate() error
}

type Notification struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Request struct {
	Notification        // embed
	ID           string `json:"id"`
}

func (js *Notification) Validate() error {
	if js.Version != RpcVersion {
		return ErrInvalidVersion
	}

	if js.Method == "" {
		return ErrMissingMethod
	}

	return nil
}

func (js *Request) Validate() error {
	if js.ID == "" {
		return ErrMissingID
	}

	return js.Notification.Validate()
}
