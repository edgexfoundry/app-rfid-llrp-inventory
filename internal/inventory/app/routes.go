//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"edgexfoundry-holding/rfid-llrp-inventory-service/internal/llrp"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	maxBodyBytes = 100 * 1024
)

func (app *InventoryApp) addRoutes() error {
	if err := app.addRoute(
		"/", http.MethodGet, app.index); err != nil {
		return err
	}
	if err := app.addRoute(
		"/api/v1/readers", http.MethodGet, app.getReaders); err != nil {
		return err
	}
	if err := app.addRoute(
		"/api/v1/inventory/snapshot", http.MethodGet, app.getSnapshot); err != nil {
		return err
	}
	if err := app.addRoute(
		"/api/v1/command/reading/start", http.MethodPost, app.startReading); err != nil {
		return err
	}
	if err := app.addRoute(
		"/api/v1/command/reading/stop", http.MethodPost, app.stopReading); err != nil {
		return err
	}
	if err := app.addRoute(
		"/api/v1/behaviors/{name}", http.MethodGet, app.getBehavior); err != nil {
		return err
	}
	if err := app.addRoute(
		"/api/v1/behaviors/{name}", http.MethodPut, app.setBehavior); err != nil {
		return err
	}

	return nil
}

func (app *InventoryApp) addRoute(path, method string, f http.HandlerFunc) error {
	if err := app.edgexSdk.AddRoute(path, f, method); err != nil {
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
		app.lc.Error("Failed to write readers list.", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}
func (app *InventoryApp) getSnapshot(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := app.requestInventorySnapshot(w); err != nil {
		app.lc.Error("Failed to write inventory snapshot.", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}
func (app *InventoryApp) startReading(w http.ResponseWriter, _ *http.Request) {
	if err := app.defaultGrp.StartAll(app.devService); err != nil {
		app.lc.Error("Failed to StartAll.", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
func (app *InventoryApp) stopReading(w http.ResponseWriter, _ *http.Request) {
	if err := app.defaultGrp.StopAll(app.devService); err != nil {
		app.lc.Error("Failed to StopAll.", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
func (app *InventoryApp) getBehavior(w http.ResponseWriter, req *http.Request) {
	rv := mux.Vars(req)
	bName := rv["name"]
	// Currently, only "default" is supported.
	if bName != "default" {
		app.lc.Error("Request to GET unknown behavior.", "name", bName)
		if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(app.defaultGrp.Behavior())
	if err != nil {
		app.lc.Error("Failed to marshal behavior.", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(data); err != nil {
		app.lc.Error("Failed to write behavior data.", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
func (app *InventoryApp) setBehavior(w http.ResponseWriter, req *http.Request) {
	rv := mux.Vars(req)
	bName := rv["name"]
	// Currently, only "default" is supported.
	if bName != "default" {
		app.lc.Error("Attempt to PUT unknown behavior.", "name", bName)
		if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := ioutil.ReadAll(io.LimitReader(req.Body, maxBodyBytes))
	if err != nil {
		app.lc.Error("Failed to read behavior body.", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var b llrp.Behavior
	if err := json.Unmarshal(data, &b); err != nil {
		app.lc.Error("Failed to unmarshal behavior body.", "error", err,
			"body", string(data))
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error())) // best effort
		return
	}

	if err := app.defaultGrp.SetBehavior(app.devService, b); err != nil {
		app.lc.Error("Failed to set new behavior.", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(err.Error())); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		return
	}

	app.lc.Info("Updated behavior.", "name", bName)
}
