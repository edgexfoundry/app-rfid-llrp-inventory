//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"bytes"
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"io"
	"net/http"
	"time"
)

const (
	httpTimeout = 60 * time.Second
)

var (
	client = &http.Client{
		Timeout: httpTimeout,
	}
)

// RawInventory returns a handler bound to the TagProcessor.
// When called, it returns the raw inventory algorithm data.
func RawInventory(lc logger.LoggingClient, tagPro *inventory.TagProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		payload, err := json.Marshal(tagPro.GetRawInventory())
		if err != nil {
			lc.Error("Failed to marshal inventory", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err = w.Write(payload); err != nil {
			lc.Error("Failed to write inventory response.", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

type Proxy struct {
	Request func() (*http.Request, error)
	LC      logger.LoggingClient
}

func NewGetProxy(logger logger.LoggingClient, endpoint string) Proxy {
	return Proxy{
		LC: logger,
		Request: func() (*http.Request, error) {
			return http.NewRequest(http.MethodGet, endpoint, nil)
		},
	}
}

func NewPutProxy(logger logger.LoggingClient, endpoint string, body []byte) Proxy {
	return Proxy{
		LC: logger,
		Request: func() (*http.Request, error) {
			return http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
		},
	}
}

func (p Proxy) HandleRequest(w http.ResponseWriter, _ *http.Request) {
	p.LC.Debug("Handling new proxy request.")

	req, err := p.Request()
	if err != nil {
		p.LC.Error("Failed to construct proxy request.", "error", err.Error())
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		p.LC.Error("Failed to send proxy request.", "error", err.Error())
		return
	}

	defer resp.Body.Close()
	logs := []interface{}{"status", resp.StatusCode}

	// Best effort: see if the body has anything useful.
	body := make([]byte, 100)
	switch n, err := io.ReadFull(resp.Body, body); err {
	case io.EOF: // no response body
	case io.ErrUnexpectedEOF:
		logs = append(logs, "response", string(body[:n]))
	case nil:
		logs = append(logs, "response (truncated)", string(body[:100]))
	default:
		logs = append(logs, "response", "<read failed: "+err.Error()+">")
	}

	if 200 <= resp.StatusCode && resp.StatusCode < 300 {
		w.WriteHeader(http.StatusNoContent)
	} else {
		p.LC.Error("Upstream request failed.", logs...)
		w.WriteHeader(http.StatusBadGateway)
		if _, err := w.Write([]byte("Upstream server failed.")); err != nil {
			p.LC.Error("Failed to write response.", "error", err.Error())
		}
	}

	p.LC.Debug("Request processed.", logs...)
}

// SetBehaviors sends command to set/apply behavior command
func SetBehaviors() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
	}
}
