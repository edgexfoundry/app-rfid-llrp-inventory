//
// Copyright (c) 2020 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/app-functions-sdk-go/pkg/transforms"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
)

const (
	serviceKey = "rfid-inventory"
)

type inventoryApp struct {
	log logger.LoggingClient
	edgexSdk  *appsdk.AppFunctionsSDK
	processor *inventory.TagProcessor
	readChnl  chan inventory.Gen2Read
	eventChnl chan inventory.Event
	done      chan bool
}

var app inventoryApp

func main() {

	app = inventoryApp{}
	app.edgexSdk = &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := app.edgexSdk.Initialize(); err != nil {
		message := fmt.Sprintf("SDK initialization failed: %v\n", err)
		if app.edgexSdk.LoggingClient != nil {
			app.edgexSdk.LoggingClient.Error(message)
		} else {
			fmt.Println(message)
		}
		os.Exit(-1)
	}
	app.log = app.edgexSdk.LoggingClient
	//app.log = logger.NewClientStdOut(serviceKey, false, "INFO")
	app.done = make(chan bool)
	app.readChnl = make(chan inventory.Gen2Read, 50)
	app.eventChnl = make(chan inventory.Event, 10)
	app.processor = inventory.NewTagProcessor(app.log)
	app.log.Info(fmt.Sprintf("Running"))

	// access the application's specific configuration settings.
	deviceNames, err := app.edgexSdk.GetAppSettingStrings("DeviceNames")
	if err != nil {
		app.log.Error(err.Error())
		os.Exit(-1)
	}
	app.log.Info(fmt.Sprintf("Filtering for devices %v", deviceNames))

	// 3) This is our pipeline configuration, the collection of functions to
	// execute every time an event is triggered.
	err = app.edgexSdk.SetFunctionsPipeline(
		transforms.NewFilter(deviceNames).FilterByDeviceName,
		inboundCoreDatae,
	)

	if err != nil {
		app.log.Error("Error in the pipeline: ", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go app.processReadChannel(&wg)
	wg.Add(1)
	go app.processEventChannel(&wg)

	// 4) Lastly, we'll go ahead and tell the SDK to "start" and begin listening for events
	// to trigger the pipeline.
	err = app.edgexSdk.MakeItRun()
	if err != nil {
		app.log.Error("MakeItRun returned error: ", err.Error())
		os.Exit(-1)
	}

	app.log.Info("waiting for channels to finish")
	app.done <- true
	app.done <- true
	wg.Wait()

	// Do any required cleanup here
	os.Exit(0)
}

func inboundCoreDatae(edgexCtx *appcontext.Context, params ...interface{}) (bool, interface{}) {

	if len(params) < 1 {
		return false, nil
	}

	if event, ok := params[0].(models.Event); ok {
		for _, reading := range event.Readings {
			if gen2Read, err := marshallGen2Read(reading); err == nil {
				app.readChnl <- gen2Read
			}
		}
	}

	// todo: this should be moved to send events, need another pipeline?
	// edgexcontext.Complete([]byte(params[0].(string)))
	return false, nil
}

var tagSerialCounter uint32

func marshallGen2Read(xevent models.Reading) (r inventory.Gen2Read, err error) {
	serial := atomic.AddUint32(&tagSerialCounter, 1) % 20
	r = inventory.Gen2Read{
		Epc:       fmt.Sprintf("EPC%06d", serial),
		Tid:       fmt.Sprintf("TID%06d", serial),
		User:      fmt.Sprintf("USR%06d", serial),
		Reserved:  fmt.Sprintf("RES%06d", serial),
		DeviceId:  xevent.Device,
		AntennaId: 0,
		Timestamp: inventory.UnixMilliNow(),
		Rssi:      450,
	}
	return
}

func (app *inventoryApp) processReadChannel(wg *sync.WaitGroup) {
	defer wg.Done()
	app.log.Info("starting read channel processing")
	for {
		select {
		case <-app.done:
			app.log.Info("exiting read channel processing")
			return
		case r := <-app.readChnl:
			app.handleGen2Read(&r)
		}
	}
}

func (app *inventoryApp) handleGen2Read(read *inventory.Gen2Read) {
	app.log.Info(fmt.Sprintf("handleGen2Read from %s", read.DeviceId))
	e := app.processor.ProcessReadData(read)
	switch e.(type) {
	case inventory.Arrived:
		app.eventChnl <- e
	case inventory.Moved:
		app.eventChnl <- e
	}

}

func (app *inventoryApp) processEventChannel(wg *sync.WaitGroup) {
	defer wg.Done()
	app.log.Info("starting event channel processing")
	for {
		select {
		case <- app.done:
			app.log.Info("exiting event channel processing")
			return
		case e := <-app.eventChnl:
			app.log.Info(fmt.Sprintf("processing event %s", e.OfType()))
		}
	}
}
