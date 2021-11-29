//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"edgexfoundry/app-rfid-llrp-inventory/internal/llrp"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	maxBodyBytes   = 100 * 1024
	readersRoute   = common.ApiBase + "/readers"
	snapshotRoute  = common.ApiBase + "/inventory/snapshot"
	cmdStartRoute  = common.ApiBase + "/command/reading/start"
	cmdStopRoute   = common.ApiBase + "/command/reading/stop"
	behaviorsRoute = common.ApiBase + "/behaviors/{name}"
)

func (app *InventoryApp) addRoutes() error {
	if err := app.addRoute(
		"/", http.MethodGet, app.index); err != nil {
		return err
	}
	if err := app.addRoute(
		readersRoute, http.MethodGet, app.getReaders); err != nil {
		return err
	}
	if err := app.addRoute(
		snapshotRoute, http.MethodGet, app.getSnapshot); err != nil {
		return err
	}
	if err := app.addRoute(
		cmdStartRoute, http.MethodPost, app.startReading); err != nil {
		return err
	}
	if err := app.addRoute(
		cmdStopRoute, http.MethodPost, app.stopReading); err != nil {
		return err
	}
	if err := app.addRoute(
		behaviorsRoute, http.MethodGet, app.getBehavior); err != nil {
		return err
	}
	if err := app.addRoute(
		behaviorsRoute, http.MethodPut, app.setBehavior); err != nil {
		return err
	}

	return nil
}

func (app *InventoryApp) addRoute(path, method string, f http.HandlerFunc) error {
	if err := app.service.AddRoute(path, f, method); err != nil {
		return errors.Wrapf(err, "failed to add route, path=%s, method=%s", path, method)
	}
	return nil
}

// Routes
func (app *InventoryApp) index(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "static/html/index.html")
}

func (app *InventoryApp) getReaders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := app.defaultGrp.WriteReaders(w); err != nil {
		msg := fmt.Sprintf("Failed to write readers list: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func (app *InventoryApp) getSnapshot(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := app.requestInventorySnapshot(w); err != nil {
		msg := fmt.Sprintf("Failed to write inventory snapshot: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func (app *InventoryApp) startReading(w http.ResponseWriter, _ *http.Request) {
	if err := app.defaultGrp.StartAll(app.devService); err != nil {
		msg := fmt.Sprintf("Failed to StartAll: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func (app *InventoryApp) stopReading(w http.ResponseWriter, _ *http.Request) {
	if err := app.defaultGrp.StopAll(app.devService); err != nil {
		msg := fmt.Sprintf("Failed to StopAll: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}

func (app *InventoryApp) getBehavior(w http.ResponseWriter, req *http.Request) {
	rv := mux.Vars(req)
	bName := rv["name"]
	// Currently, only "default" is supported.
	if bName != "default" {
		msg := fmt.Sprintf("Request to GET unknown behavior. Name: %v", bName)
		app.lc.Error(msg)
		if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		w.WriteHeader(http.StatusNotFound)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	data, err := json.Marshal(app.defaultGrp.Behavior())
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal behavior: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(data); err != nil {
		msg := fmt.Sprintf("Failed to write behavior data: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func (app *InventoryApp) setBehavior(w http.ResponseWriter, req *http.Request) {
	rv := mux.Vars(req)
	bName := rv["name"]
	// Currently, only "default" is supported.
	if bName != "default" {
		msg := fmt.Sprintf("Attempt to PUT unknown behavior. Name %v", bName)
		app.lc.Error(msg)
		if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		w.WriteHeader(http.StatusNotFound)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	data, err := ioutil.ReadAll(io.LimitReader(req.Body, maxBodyBytes))
	if err != nil {
		msg := fmt.Sprintf("Failed to read behavior data: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	var b llrp.Behavior
	if err := json.Unmarshal(data, &b); err != nil {
		msg := fmt.Sprintf("Failed to unmarshal behavior data: %v. Body: %s", err, string(data))
		app.lc.Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := app.defaultGrp.SetBehavior(app.devService, b); err != nil {
		msg := fmt.Sprintf("Failed to set net behavior: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(err.Error())); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	app.lc.Info("Updated behavior.", "name", bName)
}
