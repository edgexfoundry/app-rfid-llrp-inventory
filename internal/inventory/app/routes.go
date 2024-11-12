//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"

	"edgexfoundry/app-rfid-llrp-inventory/internal/llrp"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
)

const (
	maxBodyBytes   = 100 * 1024
	readersRoute   = common.ApiBase + "/readers"
	snapshotRoute  = common.ApiBase + "/inventory/snapshot"
	cmdStartRoute  = common.ApiBase + "/command/reading/start"
	cmdStopRoute   = common.ApiBase + "/command/reading/stop"
	behaviorsRoute = common.ApiBase + "/behaviors/:name"
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

func (app *InventoryApp) addRoute(path, method string, f echo.HandlerFunc) error {
	if err := app.service.AddCustomRoute(path, false, f, method); err != nil {
		return fmt.Errorf("failed to add route, path=%s, method=%s: %w", path, method, err)
	}
	return nil
}

// Routes
func (app *InventoryApp) index(ctx echo.Context) error {
	http.ServeFile(ctx.Response().Writer, ctx.Request(), "static/html/index.html")
	return nil
}

func (app *InventoryApp) getReaders(ctx echo.Context) error {
	w := ctx.Response().Writer
	w.Header().Set("Content-Type", "application/json")
	if err := app.defaultGrp.WriteReaders(w); err != nil {
		msg := fmt.Sprintf("Failed to write readers list: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)
	}
	return nil
}

func (app *InventoryApp) getSnapshot(ctx echo.Context) error {
	w := ctx.Response().Writer
	w.Header().Set("Content-Type", "application/json")
	if err := app.requestInventorySnapshot(w); err != nil {
		msg := fmt.Sprintf("Failed to write inventory snapshot: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)
	}
	return nil
}

func (app *InventoryApp) startReading(ctx echo.Context) error {
	if err := app.defaultGrp.StartAll(app.devService); err != nil {
		msg := fmt.Sprintf("Failed to StartAll: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)
	}
	return nil
}

func (app *InventoryApp) stopReading(ctx echo.Context) error {
	if err := app.defaultGrp.StopAll(app.devService); err != nil {
		msg := fmt.Sprintf("Failed to StopAll: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)
	}
	return nil
}

func (app *InventoryApp) getBehavior(ctx echo.Context) error {
	w := ctx.Response().Writer
	bName := ctx.Param("name")
	// Currently, only "default" is supported.
	if bName != "default" {
		msg := fmt.Sprintf("Request to GET unknown behavior. Name: %v", bName)
		app.lc.Error(msg)

		if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		return ctx.String(http.StatusNotFound, msg)
	}

	data, err := json.Marshal(app.defaultGrp.Behavior())
	if err != nil {
		msg := fmt.Sprintf("Failed to marshal behavior: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)
	}

	if _, err := w.Write(data); err != nil {
		msg := fmt.Sprintf("Failed to write behavior data: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)

	}
	return nil
}

func (app *InventoryApp) setBehavior(ctx echo.Context) error {
	w := ctx.Response().Writer
	req := ctx.Request()
	bName := ctx.Param("name")
	// Currently, only "default" is supported.
	if bName != "default" {
		msg := fmt.Sprintf("Attempt to PUT unknown behavior. Name %v", bName)
		app.lc.Error(msg)
		if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		return ctx.String(http.StatusNotFound, msg)
	}

	data, err := io.ReadAll(io.LimitReader(req.Body, maxBodyBytes))
	if err != nil {
		msg := fmt.Sprintf("Failed to read behavior data: %v", err)
		app.lc.Error(msg)
		return ctx.String(http.StatusInternalServerError, msg)
	}

	var b llrp.Behavior
	if err := json.Unmarshal(data, &b); err != nil {
		msg := fmt.Sprintf("Failed to unmarshal behavior data: %v. Body: %s", err, string(data))
		app.lc.Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		return ctx.String(http.StatusInternalServerError, msg)
	}

	if err := app.defaultGrp.SetBehavior(app.devService, b); err != nil {
		msg := fmt.Sprintf("Failed to set net behavior: %v", err)
		app.lc.Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(err.Error())); err != nil {
			app.lc.Error("Error writing failure response.", "error", err)
		}
		return ctx.String(http.StatusInternalServerError, msg)
	}

	app.lc.Info("Updated behavior.", "name", bName)
	return nil
}
