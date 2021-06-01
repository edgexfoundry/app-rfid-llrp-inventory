package llrp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func readerGroupHelper() *ReaderGroup {
	return &ReaderGroup{mu: sync.RWMutex{}, readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{
		GPITrigger: nil, ImpinjOptions: &ImpinjOptions{SuppressMonza: false}, ScanType: ScanNormal, Duration: 0, Power: PowerTarget{Max: 3000}, Frequencies: nil}}
}

func addReaderHelper(t *testing.T) (*ReaderGroup, DSClient, func()) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type Reading struct {
			Name, Value string
		}
		type edgexResp struct {
			Readings []Reading
		}
		resp2 := edgexResp{Readings: []Reading{{Name: capReadingName, Value: capabilities}}}
		b2, err := json.Marshal(resp2)
		_, err = w.Write(b2)
		require.NoError(t, err)
	}))
	client := ts.Client()

	actualURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	ds := NewDSClient(actualURL, client)
	rg := readerGroupHelper()
	err = rg.AddReader(ds, "test")
	require.NoError(t, err)
	return rg, ds, func() { ts.Close() }
}

func TestNewReaderGroup(t *testing.T) {
	tests := []struct {
		name string
		want *ReaderGroup
	}{
		{
			name: "OK",
			want: readerGroupHelper(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewReaderGroup())
		})
	}
}

func TestBehavior(t *testing.T) {
	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	tests := []struct {
		name   string
		fields fields
		want   Behavior
	}{
		{
			name:   "OK - default behavior",
			fields: fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			want:   Behavior{},
		},
		{
			name:   "OK - different behavior",
			fields: fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{GPITrigger: nil, ImpinjOptions: &ImpinjOptions{SuppressMonza: false}, ScanType: ScanDeep, Duration: 0, Power: PowerTarget{Max: 3000}, Frequencies: nil}},
			want:   Behavior{GPITrigger: nil, ImpinjOptions: &ImpinjOptions{SuppressMonza: false}, ScanType: ScanDeep, Duration: 0, Power: PowerTarget{Max: 3000}, Frequencies: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			assert.Equal(t, rg.Behavior(), tt.want)
		})
	}
}

func TestWriteReaders(t *testing.T) {
	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	tests := []struct {
		name    string
		fields  fields
		wantW   string
		wantErr error
	}{
		{
			name:    "OK - default readers",
			fields:  fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			wantW:   "{\"Readers\":[]}\n",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			w := &bytes.Buffer{}
			err := rg.WriteReaders(w)
			assert.Equal(t, err, tt.wantErr)
			assert.Equal(t, w.String(), tt.wantW)
		})
	}
}

func TestProcessTagReport(t *testing.T) {
	rg, _, tsClose := addReaderHelper(t)
	defer tsClose()

	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	type args struct {
		name string
		tags []TagReportData
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "OK - Reader found",
			fields: fields{readers: map[string]TagReader{"test": rg.readers["test"]}, env: Environment{}, behavior: Behavior{}},
			args:   args{name: "test", tags: []TagReportData{}},
			want:   true,
		},
		{
			name:   "OK - Reader not found",
			fields: fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			args:   args{name: "test", tags: []TagReportData{}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			assert.Equal(t, rg.ProcessTagReport(tt.args.name, tt.args.tags), tt.want)
		})
	}
}

func TestRemoveReader(t *testing.T) {
	rg, _, tsClose := addReaderHelper(t)
	defer tsClose()

	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "OK - remove reader success",
			fields: fields{readers: map[string]TagReader{"test": rg.readers["test"]}, env: Environment{}, behavior: Behavior{}},
			args:   args{name: "test"},
		},
		{
			name:   "OK - remove invalid reader success",
			fields: fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			args:   args{name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			rg.RemoveReader(tt.args.name)
			assert.Equal(t, rg.readers[tt.args.name], nil)
		})
	}
}

func TestAddReader(t *testing.T) {
	_, dsClient, tsClose := addReaderHelper(t)
	defer tsClose()

	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	type args struct {
		ds   DSClient
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name:    "OK - add reader success",
			fields:  fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{Power: PowerTarget{Max: 30000}}},
			args:    args{ds: dsClient, name: "test-reader"},
			wantErr: nil,
		},
		{
			name:    "OK - behavior cannot be satisfied",
			fields:  fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			args:    args{ds: dsClient, name: "test-reader"},
			wantErr: ErrUnsatisfiable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			err := rg.AddReader(tt.args.ds, tt.args.name)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func Test_replaceRO(t *testing.T) {
	rg, dsClient, tsClose := addReaderHelper(t)
	defer tsClose()

	s, err := rg.readers["test"].NewROSpec(Behavior{Power: PowerTarget{Max: 30000}}, Environment{})
	assert.NoError(t, err)

	type args struct {
		ds   DSClient
		name string
		spec *ROSpec
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name:    "OK - defaults",
			args:    args{ds: dsClient, name: "", spec: nil},
			wantErr: nil,
		},
		{
			name:    "OK",
			args:    args{ds: dsClient, name: "test", spec: s},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := replaceRO(tt.args.ds, tt.args.name, tt.args.spec)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSetBehavior(t *testing.T) {
	rg, dsClient, tsClose := addReaderHelper(t)
	defer tsClose()

	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	type args struct {
		ds DSClient
		b  Behavior
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name:    "OK - defaults",
			fields:  fields{readers: nil, env: Environment{}, behavior: Behavior{}},
			args:    args{ds: dsClient, b: Behavior{}},
			wantErr: nil,
		},
		{
			name:    "OK - change to valid behavior",
			fields:  fields{readers: nil, env: Environment{}, behavior: Behavior{Power: PowerTarget{3000}}},
			args:    args{ds: dsClient, b: Behavior{Power: PowerTarget{3000}}},
			wantErr: nil,
		},
		{
			name:    "OK - attempt change to invalid behavior",
			fields:  fields{readers: map[string]TagReader{"test": rg.readers["test"]}, env: Environment{}, behavior: Behavior{Power: PowerTarget{-1}, GPITrigger: &GPITrigger{Port: 0, Event: false, Timeout: ImpinjSearchMode}}},
			args:    args{ds: dsClient, b: Behavior{Power: PowerTarget{-1}, GPITrigger: &GPITrigger{Port: 0, Event: false, Timeout: ImpinjSearchMode}}},
			wantErr: ErrUnsatisfiable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			err := rg.SetBehavior(tt.args.ds, tt.args.b)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name string
		me   MultiErr
		want string
	}{
		{
			name: "OK - no errors",
			me:   MultiErr{},
			want: "",
		},
		{
			name: "OK - one error",
			me:   MultiErr{ErrUnsatisfiable},
			want: fmt.Sprintf("%v", ErrUnsatisfiable.Error()),
		},
		{
			name: "OK - two errors",
			me:   MultiErr{ErrUnsatisfiable, ErrMissingCapInfo},
			want: fmt.Sprintf("%v; %v", ErrUnsatisfiable.Error(), ErrMissingCapInfo.Error()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.me.Error()
			require.Equal(t, got, tt.want)
		})
	}
}

func TestStartAll(t *testing.T) {
	rg, dsClient, tsClose := addReaderHelper(t)
	defer tsClose()

	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	type args struct {
		ds DSClient
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name:    "OK - no tag readers",
			fields:  fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			args:    args{ds: dsClient},
			wantErr: nil,
		},
		{
			name:    "OK - tag readers",
			fields:  fields{readers: map[string]TagReader{"test": rg.readers["test"]}, env: Environment{}, behavior: Behavior{}},
			args:    args{ds: dsClient},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			err := rg.StartAll(tt.args.ds)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestStopAll(t *testing.T) {
	rg, dsClient, tsClose := addReaderHelper(t)
	defer tsClose()

	type fields struct {
		readers  map[string]TagReader
		env      Environment
		behavior Behavior
	}
	type args struct {
		ds DSClient
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name:    "OK - no tag readers",
			fields:  fields{readers: map[string]TagReader{}, env: Environment{}, behavior: Behavior{}},
			args:    args{ds: dsClient},
			wantErr: nil,
		},
		{
			name:    "OK - tag readers",
			fields:  fields{readers: map[string]TagReader{"test": rg.readers["test"]}, env: Environment{}, behavior: Behavior{}},
			args:    args{ds: dsClient},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &ReaderGroup{
				readers:  tt.fields.readers,
				env:      tt.fields.env,
				behavior: tt.fields.behavior,
			}
			err := rg.StartAll(tt.args.ds)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

const capabilities = `{
	"LLRPStatus": {
		"Status": 0,
		"ErrorDescription": "",
		"FieldError": null,
		"ParameterError": null
	},
	"GeneralDeviceCapabilities": {
		"MaxSupportedAntennas": 4,
		"CanSetAntennaProperties": false,
		"HasUTCClock": true,
		"DeviceManufacturer": 25882,
		"Model": 2001002,
		"FirmwareVersion": "5.14.0.240",
		"ReceiveSensitivities": [
			{
				"Index": 1,
				"ReceiveSensitivity": 0
			},
			{
				"Index": 2,
				"ReceiveSensitivity": 10
			},
			{
				"Index": 3,
				"ReceiveSensitivity": 11
			},
			{
				"Index": 4,
				"ReceiveSensitivity": 12
			},
			{
				"Index": 5,
				"ReceiveSensitivity": 13
			},
			{
				"Index": 6,
				"ReceiveSensitivity": 14
			},
			{
				"Index": 7,
				"ReceiveSensitivity": 15
			},
			{
				"Index": 8,
				"ReceiveSensitivity": 16
			},
			{
				"Index": 9,
				"ReceiveSensitivity": 17
			},
			{
				"Index": 10,
				"ReceiveSensitivity": 18
			},
			{
				"Index": 11,
				"ReceiveSensitivity": 19
			},
			{
				"Index": 12,
				"ReceiveSensitivity": 20
			},
			{
				"Index": 13,
				"ReceiveSensitivity": 21
			},
			{
				"Index": 14,
				"ReceiveSensitivity": 22
			},
			{
				"Index": 15,
				"ReceiveSensitivity": 23
			},
			{
				"Index": 16,
				"ReceiveSensitivity": 24
			},
			{
				"Index": 17,
				"ReceiveSensitivity": 25
			},
			{
				"Index": 18,
				"ReceiveSensitivity": 26
			},
			{
				"Index": 19,
				"ReceiveSensitivity": 27
			},
			{
				"Index": 20,
				"ReceiveSensitivity": 28
			},
			{
				"Index": 21,
				"ReceiveSensitivity": 29
			},
			{
				"Index": 22,
				"ReceiveSensitivity": 30
			},
			{
				"Index": 23,
				"ReceiveSensitivity": 31
			},
			{
				"Index": 24,
				"ReceiveSensitivity": 32
			},
			{
				"Index": 25,
				"ReceiveSensitivity": 33
			},
			{
				"Index": 26,
				"ReceiveSensitivity": 34
			},
			{
				"Index": 27,
				"ReceiveSensitivity": 35
			},
			{
				"Index": 28,
				"ReceiveSensitivity": 36
			},
			{
				"Index": 29,
				"ReceiveSensitivity": 37
			},
			{
				"Index": 30,
				"ReceiveSensitivity": 38
			},
			{
				"Index": 31,
				"ReceiveSensitivity": 39
			},
			{
				"Index": 32,
				"ReceiveSensitivity": 40
			},
			{
				"Index": 33,
				"ReceiveSensitivity": 41
			},
			{
				"Index": 34,
				"ReceiveSensitivity": 42
			},
			{
				"Index": 35,
				"ReceiveSensitivity": 43
			},
			{
				"Index": 36,
				"ReceiveSensitivity": 44
			},
			{
				"Index": 37,
				"ReceiveSensitivity": 45
			},
			{
				"Index": 38,
				"ReceiveSensitivity": 46
			},
			{
				"Index": 39,
				"ReceiveSensitivity": 47
			},
			{
				"Index": 40,
				"ReceiveSensitivity": 48
			},
			{
				"Index": 41,
				"ReceiveSensitivity": 49
			},
			{
				"Index": 42,
				"ReceiveSensitivity": 50
			}
		],
		"PerAntennaReceiveSensitivityRanges": null,
		"GPIOCapabilities": {
			"NumGPIs": 4,
			"NumGPOs": 4
		},
		"PerAntennaAirProtocols": [
			{
				"AntennaID": 1,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 2,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 3,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 4,
				"AirProtocolIDs": "AQ=="
			}
		],
		"MaximumReceiveSensitivity": null
	},
	"LLRPCapabilities": {
		"CanDoRFSurvey": false,
		"CanReportBufferFillWarning": true,
		"SupportsClientRequestOpSpec": false,
		"CanDoTagInventoryStateAwareSingulation": false,
		"SupportsEventsAndReportHolding": true,
		"MaxPriorityLevelSupported": 1,
		"ClientRequestedOpSpecTimeout": 0,
		"MaxROSpecs": 1,
		"MaxSpecsPerROSpec": 32,
		"MaxInventoryParameterSpecsPerAISpec": 1,
		"MaxAccessSpecs": 1508,
		"MaxOpSpecsPerAccessSpec": 8
	},
	"RegulatoryCapabilities": {
		"CountryCode": 840,
		"CommunicationsStandard": 1,
		"UHFBandCapabilities": {
			"TransmitPowerLevels": [
				{
					"Index": 1,
					"TransmitPowerValue": 1000
				},
				{
					"Index": 2,
					"TransmitPowerValue": 1025
				},
				{
					"Index": 3,
					"TransmitPowerValue": 1050
				},
				{
					"Index": 4,
					"TransmitPowerValue": 1075
				},
				{
					"Index": 5,
					"TransmitPowerValue": 1100
				},
				{
					"Index": 6,
					"TransmitPowerValue": 1125
				},
				{
					"Index": 7,
					"TransmitPowerValue": 1150
				},
				{
					"Index": 8,
					"TransmitPowerValue": 1175
				},
				{
					"Index": 9,
					"TransmitPowerValue": 1200
				},
				{
					"Index": 10,
					"TransmitPowerValue": 1225
				},
				{
					"Index": 11,
					"TransmitPowerValue": 1250
				},
				{
					"Index": 12,
					"TransmitPowerValue": 1275
				},
				{
					"Index": 13,
					"TransmitPowerValue": 1300
				},
				{
					"Index": 14,
					"TransmitPowerValue": 1325
				},
				{
					"Index": 15,
					"TransmitPowerValue": 1350
				},
				{
					"Index": 16,
					"TransmitPowerValue": 1375
				},
				{
					"Index": 17,
					"TransmitPowerValue": 1400
				},
				{
					"Index": 18,
					"TransmitPowerValue": 1425
				},
				{
					"Index": 19,
					"TransmitPowerValue": 1450
				},
				{
					"Index": 20,
					"TransmitPowerValue": 1475
				},
				{
					"Index": 21,
					"TransmitPowerValue": 1500
				},
				{
					"Index": 22,
					"TransmitPowerValue": 1525
				},
				{
					"Index": 23,
					"TransmitPowerValue": 1550
				},
				{
					"Index": 24,
					"TransmitPowerValue": 1575
				},
				{
					"Index": 25,
					"TransmitPowerValue": 1600
				},
				{
					"Index": 26,
					"TransmitPowerValue": 1625
				},
				{
					"Index": 27,
					"TransmitPowerValue": 1650
				},
				{
					"Index": 28,
					"TransmitPowerValue": 1675
				},
				{
					"Index": 29,
					"TransmitPowerValue": 1700
				},
				{
					"Index": 30,
					"TransmitPowerValue": 1725
				},
				{
					"Index": 31,
					"TransmitPowerValue": 1750
				},
				{
					"Index": 32,
					"TransmitPowerValue": 1775
				},
				{
					"Index": 33,
					"TransmitPowerValue": 1800
				},
				{
					"Index": 34,
					"TransmitPowerValue": 1825
				},
				{
					"Index": 35,
					"TransmitPowerValue": 1850
				},
				{
					"Index": 36,
					"TransmitPowerValue": 1875
				},
				{
					"Index": 37,
					"TransmitPowerValue": 1900
				},
				{
					"Index": 38,
					"TransmitPowerValue": 1925
				},
				{
					"Index": 39,
					"TransmitPowerValue": 1950
				},
				{
					"Index": 40,
					"TransmitPowerValue": 1975
				},
				{
					"Index": 41,
					"TransmitPowerValue": 2000
				},
				{
					"Index": 42,
					"TransmitPowerValue": 2025
				},
				{
					"Index": 43,
					"TransmitPowerValue": 2050
				},
				{
					"Index": 44,
					"TransmitPowerValue": 2075
				},
				{
					"Index": 45,
					"TransmitPowerValue": 2100
				},
				{
					"Index": 46,
					"TransmitPowerValue": 2125
				},
				{
					"Index": 47,
					"TransmitPowerValue": 2150
				},
				{
					"Index": 48,
					"TransmitPowerValue": 2175
				},
				{
					"Index": 49,
					"TransmitPowerValue": 2200
				},
				{
					"Index": 50,
					"TransmitPowerValue": 2225
				},
				{
					"Index": 51,
					"TransmitPowerValue": 2250
				},
				{
					"Index": 52,
					"TransmitPowerValue": 2275
				},
				{
					"Index": 53,
					"TransmitPowerValue": 2300
				},
				{
					"Index": 54,
					"TransmitPowerValue": 2325
				},
				{
					"Index": 55,
					"TransmitPowerValue": 2350
				},
				{
					"Index": 56,
					"TransmitPowerValue": 2375
				},
				{
					"Index": 57,
					"TransmitPowerValue": 2400
				},
				{
					"Index": 58,
					"TransmitPowerValue": 2425
				},
				{
					"Index": 59,
					"TransmitPowerValue": 2450
				},
				{
					"Index": 60,
					"TransmitPowerValue": 2475
				},
				{
					"Index": 61,
					"TransmitPowerValue": 2500
				},
				{
					"Index": 62,
					"TransmitPowerValue": 2525
				},
				{
					"Index": 63,
					"TransmitPowerValue": 2550
				},
				{
					"Index": 64,
					"TransmitPowerValue": 2575
				},
				{
					"Index": 65,
					"TransmitPowerValue": 2600
				},
				{
					"Index": 66,
					"TransmitPowerValue": 2625
				},
				{
					"Index": 67,
					"TransmitPowerValue": 2650
				},
				{
					"Index": 68,
					"TransmitPowerValue": 2675
				},
				{
					"Index": 69,
					"TransmitPowerValue": 2700
				},
				{
					"Index": 70,
					"TransmitPowerValue": 2725
				},
				{
					"Index": 71,
					"TransmitPowerValue": 2750
				},
				{
					"Index": 72,
					"TransmitPowerValue": 2775
				},
				{
					"Index": 73,
					"TransmitPowerValue": 2800
				},
				{
					"Index": 74,
					"TransmitPowerValue": 2825
				},
				{
					"Index": 75,
					"TransmitPowerValue": 2850
				},
				{
					"Index": 76,
					"TransmitPowerValue": 2875
				},
				{
					"Index": 77,
					"TransmitPowerValue": 2900
				},
				{
					"Index": 78,
					"TransmitPowerValue": 2925
				},
				{
					"Index": 79,
					"TransmitPowerValue": 2950
				},
				{
					"Index": 80,
					"TransmitPowerValue": 2975
				},
				{
					"Index": 81,
					"TransmitPowerValue": 3000
				}
			],
			"FrequencyInformation": {
                "Hopping": true,
                "FrequencyHopTables": [
                    {
                        "HopTableID": 1,
                        "Frequencies": [
                            909250,
                            908250,
                            925750,
                            911250,
                            910750,
                            926750,
                            917750,
                            905250,
                            927250,
                            921250,
                            925250,
                            919250,
                            924750,
                            916250,
                            919750,
                            913250,
                            926250,
                            916750,
                            918750,
                            914250,
                            909750,
                            917250,
                            908750,
                            902750,
                            921750,
                            913750,
                            915750,
                            923750,
                            904250,
                            903750,
                            903250,
                            907750,
                            915250,
                            924250,
                            912750,
                            918250,
                            912250,
                            910250,
                            922250,
                            905750,
                            906750,
                            920750,
                            923250,
                            906250,
                            914750,
                            911750,
                            920250,
                            907250,
                            922750,
                            904750
                        ]
                    }
                ],
                "FixedFrequencyTable": null
            },
			"C1G2RFModes": {
				"UHFC1G2RFModeTableEntries": [
					{
						"ModeID": 0,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 1,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 2,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 2,
						"ForwardLinkModulation": 0,
						"SpectralMask": 3,
						"BackscatterDataRate": 274000,
						"PIERatio": 2000,
						"MinTariTime": 20000,
						"MaxTariTime": 20000,
						"StepTariTime": 0
					},
					{
						"ModeID": 3,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 3,
						"ForwardLinkModulation": 0,
						"SpectralMask": 3,
						"BackscatterDataRate": 170600,
						"PIERatio": 2000,
						"MinTariTime": 20000,
						"MaxTariTime": 20000,
						"StepTariTime": 0
					},
					{
						"ModeID": 4,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 2,
						"ForwardLinkModulation": 0,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 7140,
						"MaxTariTime": 7140,
						"StepTariTime": 0
					},
					{
						"ModeID": 1000,
						"DivideRatio": 0,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 0,
						"SpectralMask": 0,
						"BackscatterDataRate": 40000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1002,
						"DivideRatio": 0,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 0,
						"SpectralMask": 0,
						"BackscatterDataRate": 40000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1003,
						"DivideRatio": 0,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 0,
						"SpectralMask": 0,
						"BackscatterDataRate": 40000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1004,
						"DivideRatio": 0,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 0,
						"SpectralMask": 0,
						"BackscatterDataRate": 40000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1005,
						"DivideRatio": 0,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 0,
						"SpectralMask": 0,
						"BackscatterDataRate": 40000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					}
				]
			},
			"RFSurveyFrequencyCapabilities": null
		},
		"Custom": null
	},
	"C1G2LLRPCapabilities": {
		"SupportsBlockErase": false,
		"SupportsBlockWrite": true,
		"SupportsBlockPermalock": false,
		"SupportsTagRecommissioning": false,
		"SupportsUMIMethod2": false,
		"SupportsXPC": false,
		"MaxSelectFiltersPerQuery": 2
	},
	"Custom": null
}`
