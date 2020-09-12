//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

//go:generate stringer -type=VendorIDType,ImpinjModelType,ImpinjMessageSubtype,ImpinjParamSubtype -output types_string.go

package inventory

type VendorIDType uint32

const (
	Impinj = VendorIDType(25882)
	Alien  = VendorIDType(17996)
	Zebra  = VendorIDType(10642)
)

type ImpinjModelType uint32

const (
	SpeedwayR220 = ImpinjModelType(2001001)
	SpeedwayR420 = ImpinjModelType(2001002)
	XPortal      = ImpinjModelType(2001003)
	XArrayWM     = ImpinjModelType(2001004)
	XArrayEAP    = ImpinjModelType(2001006)
	XArray       = ImpinjModelType(2001007)
	XSpan        = ImpinjModelType(2001008)
	SpeedwayR120 = ImpinjModelType(2001009)
	R700         = ImpinjModelType(2001052)
)

type ImpinjMessageSubtype uint32

const (
	ImpinjEnableExtensions         = ImpinjMessageSubtype(21)
	ImpinjEnableExtensionsResponse = ImpinjMessageSubtype(22)

	ImpinjSaveSettings         = ImpinjMessageSubtype(23)
	ImpinjSaveSettingsResponse = ImpinjMessageSubtype(24)
)

type ImpinjParamSubtype uint32

const (
	ImpinjDetailedVersion     = ImpinjParamSubtype(29)
	ImpinjEnablePeakRSSI      = ImpinjParamSubtype(53)
	ImpinjEnableOptimizedRead = ImpinjParamSubtype(65)
	ImpinjPeakRSSI            = ImpinjParamSubtype(57)
)
