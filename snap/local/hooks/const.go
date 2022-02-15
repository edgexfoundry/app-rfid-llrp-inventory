// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2021 Canonical Ltd
 *
 *  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *
 * SPDX-License-Identifier: Apache-2.0'
 */

package hooks

// ConfToEnv defines mappings from snap config keys to EdgeX environment variable
// names that are used to override individual device-mqtt's [Driver]  configuration
// values via a .env file read by the snap service wrapper.
//
// The syntax to set a configuration key is:
//
// env.<section>.<keyname>
//
var ConfToEnv = map[string]string{

	//  [AppCustom.Aliases]
	"appcustom.appsettings.device-service-name":             "APPCUSTOM_APPSETTINGS_DEVICESERVICENAME",
	"appcustom.appsettings.adjust-last-read-on-by-origin":   "APPCUSTOM_APPSETTINGS_ADJUSTLASTREADONBYORIGIN",
	"appcustom.appsettings.departed-threshold-seconds":      "APPCUSTOM_APPSETTINGS_DEPARTEDTHRESHOLDSECONDS",
	"appcustom.appsettings.departed-check-interval-seconds": "APPCUSTOM_APPSETTINGS_DEPARTEDCHECKINTERVALSECONDS",
	"appcustom.appsettings.age-out-hours":                   "APPCUSTOM_APPSETTINGS_AGEOUTHOURS",
	"appcustom.appsettings.mobility-profile-threshold":      "APPCUSTOM_APPSETTINGS_MOBILITYPROFILETHRESHOLD",
	"appcustom.appsettings.mobility-profile-holdoff-millis": "APPCUSTOM_APPSETTINGS_MOBILITYPROFILEHOLDOFFMILLIS",
	"appcustom.appsettings.mobility-profile-slope":          "APPCUSTOM_APPSETTINGS_MOBILITYPROFILESLOPE",
}
