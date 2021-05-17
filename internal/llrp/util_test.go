//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWordsToHex(t *testing.T) {
	var tests = []struct {
		name string
		x    []uint16
		want string
	}{
		{
			name: "OK - empty",
			x:    []uint16{},
			want: "",
		},
		{
			name: "OK - range: 0-7",
			x:    []uint16{0, 1, 2, 3, 4, 5, 6, 7},
			want: "00000001000200030004000500060007",
		},
		{
			name: "OK - range: 8-15",
			x:    []uint16{8, 9, 10, 11, 12, 13, 14, 15},
			want: "00080009000a000b000c000d000e000f",
		},
		{
			name: "OK- range: f0-f7",
			x:    []uint16{0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7},
			want: "00f000f100f200f300f400f500f600f7",
		},
		{
			name: "OK - range: f8-ff",
			x:    []uint16{0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff},
			want: "00f800f900fa00fb00fc00fd00fe00ff",
		},
		{
			name: "OK - g",
			x:    []uint16{'g'},
			want: "0067",
		},
		{
			name: "OK - range: e3, a1",
			x:    []uint16{0xe3, 0xa1},
			want: "00e300a1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := wordsToHex(tt.x)
			require.Equal(t, tt.want, res)
		})
	}
}

func peakHelper(val float64) *PeakRSSI {
	peak := PeakRSSI(val)
	return &peak
}

func TestExtractRSSI(t *testing.T) {
	type fields struct {
		EPCData                                 EPCData
		EPC96                                   EPC96
		ROSpecID                                *ROSpecID
		SpecIndex                               *SpecIndex
		InventoryParameterSpecID                *InventoryParameterSpecID
		AntennaID                               *AntennaID
		PeakRSSI                                *PeakRSSI
		ChannelIndex                            *ChannelIndex
		FirstSeenUTC                            *FirstSeenUTC
		FirstSeenUptime                         *FirstSeenUptime
		LastSeenUTC                             *LastSeenUTC
		LastSeenUptime                          *LastSeenUptime
		TagSeenCount                            *TagSeenCount
		C1G2PC                                  *C1G2PC
		C1G2XPCW1                               *C1G2XPCW1
		C1G2XPCW2                               *C1G2XPCW2
		C1G2CRC                                 *C1G2CRC
		AccessSpecID                            *AccessSpecID
		C1G2ReadOpSpecResult                    *C1G2ReadOpSpecResult
		C1G2WriteOpSpecResult                   *C1G2WriteOpSpecResult
		C1G2KillOpSpecResult                    *C1G2KillOpSpecResult
		C1G2LockOpSpecResult                    *C1G2LockOpSpecResult
		C1G2BlockEraseOpSpecResult              *C1G2BlockEraseOpSpecResult
		C1G2BlockWriteOpSpecResult              *C1G2BlockWriteOpSpecResult
		C1G2RecommissionOpSpecResult            *C1G2RecommissionOpSpecResult
		C1G2BlockPermalockOpSpecResult          *C1G2BlockPermalockOpSpecResult
		C1G2GetBlockPermalockStatusOpSpecResult *C1G2GetBlockPermalockStatusOpSpecResult
		ClientRequestOpSpecResult               *ClientRequestOpSpecResult
		Custom                                  []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
		want1  bool
	}{
		{
			name:   "OK",
			fields: fields{PeakRSSI: new(PeakRSSI)},
			want:   float64(0),
			want1:  true,
		},
		{
			name:   "OK - peak value",
			fields: fields{PeakRSSI: peakHelper(3)},
			want:   float64(3),
			want1:  true,
		},
		{
			name:   "OK - nil",
			fields: fields{PeakRSSI: nil},
			want:   float64(0),
			want1:  false,
		},
		{
			name:   "OK - custom",
			fields: fields{Custom: []Custom{{VendorID: uint32(PENImpinj), Subtype: ImpinjEnablePeakRSSI, Data: []byte{'1', '2'}}}},
			want:   float64(int16(binary.BigEndian.Uint16([]byte{'1', '2'}))) / 100.0,
			want1:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &TagReportData{
				EPCData:                                 tt.fields.EPCData,
				EPC96:                                   tt.fields.EPC96,
				ROSpecID:                                tt.fields.ROSpecID,
				SpecIndex:                               tt.fields.SpecIndex,
				InventoryParameterSpecID:                tt.fields.InventoryParameterSpecID,
				AntennaID:                               tt.fields.AntennaID,
				PeakRSSI:                                tt.fields.PeakRSSI,
				ChannelIndex:                            tt.fields.ChannelIndex,
				FirstSeenUTC:                            tt.fields.FirstSeenUTC,
				FirstSeenUptime:                         tt.fields.FirstSeenUptime,
				LastSeenUTC:                             tt.fields.LastSeenUTC,
				LastSeenUptime:                          tt.fields.LastSeenUptime,
				TagSeenCount:                            tt.fields.TagSeenCount,
				C1G2PC:                                  tt.fields.C1G2PC,
				C1G2XPCW1:                               tt.fields.C1G2XPCW1,
				C1G2XPCW2:                               tt.fields.C1G2XPCW2,
				C1G2CRC:                                 tt.fields.C1G2CRC,
				AccessSpecID:                            tt.fields.AccessSpecID,
				C1G2ReadOpSpecResult:                    tt.fields.C1G2ReadOpSpecResult,
				C1G2WriteOpSpecResult:                   tt.fields.C1G2WriteOpSpecResult,
				C1G2KillOpSpecResult:                    tt.fields.C1G2KillOpSpecResult,
				C1G2LockOpSpecResult:                    tt.fields.C1G2LockOpSpecResult,
				C1G2BlockEraseOpSpecResult:              tt.fields.C1G2BlockEraseOpSpecResult,
				C1G2BlockWriteOpSpecResult:              tt.fields.C1G2BlockWriteOpSpecResult,
				C1G2RecommissionOpSpecResult:            tt.fields.C1G2RecommissionOpSpecResult,
				C1G2BlockPermalockOpSpecResult:          tt.fields.C1G2BlockPermalockOpSpecResult,
				C1G2GetBlockPermalockStatusOpSpecResult: tt.fields.C1G2GetBlockPermalockStatusOpSpecResult,
				ClientRequestOpSpecResult:               tt.fields.ClientRequestOpSpecResult,
				Custom:                                  tt.fields.Custom,
			}
			got, got1 := rt.ExtractRSSI()
			require.Equal(t, got, tt.want)
			require.Equal(t, got1, tt.want1)
		})
	}
}

func helper() *C1G2ReadOpSpecResult {
	val := C1G2ReadOpSpecResult{C1G2ReadOpSpecResultType: 1, OpSpecID: impjDualTarget, Data: []uint16{1, 2}}
	return &val
}

func TestReadDataAsHex(t *testing.T) {
	type fields struct {
		EPCData                                 EPCData
		EPC96                                   EPC96
		ROSpecID                                *ROSpecID
		SpecIndex                               *SpecIndex
		InventoryParameterSpecID                *InventoryParameterSpecID
		AntennaID                               *AntennaID
		PeakRSSI                                *PeakRSSI
		ChannelIndex                            *ChannelIndex
		FirstSeenUTC                            *FirstSeenUTC
		FirstSeenUptime                         *FirstSeenUptime
		LastSeenUTC                             *LastSeenUTC
		LastSeenUptime                          *LastSeenUptime
		TagSeenCount                            *TagSeenCount
		C1G2PC                                  *C1G2PC
		C1G2XPCW1                               *C1G2XPCW1
		C1G2XPCW2                               *C1G2XPCW2
		C1G2CRC                                 *C1G2CRC
		AccessSpecID                            *AccessSpecID
		C1G2ReadOpSpecResult                    *C1G2ReadOpSpecResult
		C1G2WriteOpSpecResult                   *C1G2WriteOpSpecResult
		C1G2KillOpSpecResult                    *C1G2KillOpSpecResult
		C1G2LockOpSpecResult                    *C1G2LockOpSpecResult
		C1G2BlockEraseOpSpecResult              *C1G2BlockEraseOpSpecResult
		C1G2BlockWriteOpSpecResult              *C1G2BlockWriteOpSpecResult
		C1G2RecommissionOpSpecResult            *C1G2RecommissionOpSpecResult
		C1G2BlockPermalockOpSpecResult          *C1G2BlockPermalockOpSpecResult
		C1G2GetBlockPermalockStatusOpSpecResult *C1G2GetBlockPermalockStatusOpSpecResult
		ClientRequestOpSpecResult               *ClientRequestOpSpecResult
		Custom                                  []Custom
	}

	tests := []struct {
		name     string
		fields   fields
		wantData string
		wantOk   bool
	}{
		{
			name:     "OK - nil",
			fields:   fields{C1G2ReadOpSpecResult: nil},
			wantData: "",
			wantOk:   false,
		},
		{
			name:     "OK - default values",
			fields:   fields{C1G2ReadOpSpecResult: new(C1G2ReadOpSpecResult)},
			wantData: wordsToHex([]uint16{}),
			wantOk:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &TagReportData{
				EPCData:                                 tt.fields.EPCData,
				EPC96:                                   tt.fields.EPC96,
				ROSpecID:                                tt.fields.ROSpecID,
				SpecIndex:                               tt.fields.SpecIndex,
				InventoryParameterSpecID:                tt.fields.InventoryParameterSpecID,
				AntennaID:                               tt.fields.AntennaID,
				PeakRSSI:                                tt.fields.PeakRSSI,
				ChannelIndex:                            tt.fields.ChannelIndex,
				FirstSeenUTC:                            tt.fields.FirstSeenUTC,
				FirstSeenUptime:                         tt.fields.FirstSeenUptime,
				LastSeenUTC:                             tt.fields.LastSeenUTC,
				LastSeenUptime:                          tt.fields.LastSeenUptime,
				TagSeenCount:                            tt.fields.TagSeenCount,
				C1G2PC:                                  tt.fields.C1G2PC,
				C1G2XPCW1:                               tt.fields.C1G2XPCW1,
				C1G2XPCW2:                               tt.fields.C1G2XPCW2,
				C1G2CRC:                                 tt.fields.C1G2CRC,
				AccessSpecID:                            tt.fields.AccessSpecID,
				C1G2ReadOpSpecResult:                    tt.fields.C1G2ReadOpSpecResult,
				C1G2WriteOpSpecResult:                   tt.fields.C1G2WriteOpSpecResult,
				C1G2KillOpSpecResult:                    tt.fields.C1G2KillOpSpecResult,
				C1G2LockOpSpecResult:                    tt.fields.C1G2LockOpSpecResult,
				C1G2BlockEraseOpSpecResult:              tt.fields.C1G2BlockEraseOpSpecResult,
				C1G2BlockWriteOpSpecResult:              tt.fields.C1G2BlockWriteOpSpecResult,
				C1G2RecommissionOpSpecResult:            tt.fields.C1G2RecommissionOpSpecResult,
				C1G2BlockPermalockOpSpecResult:          tt.fields.C1G2BlockPermalockOpSpecResult,
				C1G2GetBlockPermalockStatusOpSpecResult: tt.fields.C1G2GetBlockPermalockStatusOpSpecResult,
				ClientRequestOpSpecResult:               tt.fields.ClientRequestOpSpecResult,
				Custom:                                  tt.fields.Custom,
			}
			gotData, gotOk := rt.ReadDataAsHex()
			require.Equal(t, gotData, tt.wantData)
			require.Equal(t, gotOk, tt.wantOk)
		})
	}
}

func TestIs(t *testing.T) {
	type fields struct {
		VendorID uint32
		Subtype  uint32
		Data     []byte
	}
	type args struct {
		idType  VendorPEN
		subtype CustomParamSubtype
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "OK",
			fields: fields{VendorID: uint32(PENImpinj), Subtype: ImpinjTagReportContentSelector, Data: impinjEnableBool16(ImpinjSearchMode)},
			args:   args{idType: PENImpinj, subtype: ImpinjTagReportContentSelector},
			want:   true,
		},
		{
			name:   "OK - mismatched subtype",
			fields: fields{VendorID: uint32(PENImpinj), Subtype: ImpinjTagReportContentSelector, Data: impinjEnableBool16(ImpinjEnablePeakRSSI)},
			args:   args{idType: PENImpinj, subtype: ImpinjPeakRSSI},
			want:   false,
		},
		{
			name:   "OK - mismatched idType",
			fields: fields{VendorID: uint32(PENImpinj), Subtype: ImpinjTagReportContentSelector, Data: impinjEnableBool16(ImpinjEnablePeakRSSI)},
			args:   args{idType: PENAlien, subtype: ImpinjTagReportContentSelector},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Custom{
				VendorID: tt.fields.VendorID,
				Subtype:  tt.fields.Subtype,
				Data:     tt.fields.Data,
			}
			require.Equal(t, c.Is(tt.args.idType, tt.args.subtype), tt.want)
		})
	}
}
