package inventory

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func tagHelper(epc string, state TagState) *Tag {
	return &Tag{EPC: "test", state: TagState(state), statsMap: make(map[string]*tagStats)}
}

func TestNewTag(t *testing.T) {
	type args struct {
		epc string
	}
	tests := []struct {
		name string
		args args
		want *Tag
	}{
		{
			name: "OK",
			args: args{epc: "test"},
			want: tagHelper("test", TagState(Unknown)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTag(tt.args.epc)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestSetState(t *testing.T) {
	type fields struct {
		LastRead int64
		state    TagState
	}
	type args struct {
		newState TagState
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "OK - change to Unknown state",
			fields: fields{state: TagState(Present), LastRead: 9999999999},
			args:   args{newState: TagState(Unknown)},
		},
		{
			name:   "OK - change to Departed state",
			fields: fields{state: TagState(Present), LastRead: 9999999999},
			args:   args{newState: TagState(Departed)},
		},
		{
			name:   "OK - maintain Unknown state",
			fields: fields{state: TagState(Unknown), LastRead: 9999999999},
			args:   args{newState: TagState(Unknown)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := &Tag{
				LastRead: tt.fields.LastRead,
				state:    tt.fields.state,
			}
			tag.setState(tt.args.newState)
			assert.Equal(t, tt.args.newState, tag.state)
		})
	}
}

func TestSetStateAt(t *testing.T) {
	type fields struct {
		LastDeparted int64
		LastArrived  int64
		state        TagState
	}
	type args struct {
		newState  TagState
		timestamp int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "OK - change to Present state",
			fields: fields{state: Unknown, LastArrived: 9999999998, LastDeparted: 9999999999},
			args:   args{newState: Present, timestamp: 9999999999},
		},
		{
			name:   "OK - change to Departed state",
			fields: fields{state: Unknown, LastDeparted: 9999999998, LastArrived: 9999999999},
			args:   args{newState: Departed, timestamp: 9999999999},
		},
		{
			name:   "OK - change to Unknown state",
			fields: fields{state: Departed, LastDeparted: 9999999998, LastArrived: 9999999998},
			args:   args{newState: Unknown, timestamp: 9999999998},
		},
		{
			name:   "OK - maintain Unknown state",
			fields: fields{state: Unknown, LastDeparted: 9999999998, LastArrived: 9999999998},
			args:   args{newState: Unknown, timestamp: 9999999998},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := &Tag{
				LastDeparted: tt.fields.LastDeparted,
				LastArrived:  tt.fields.LastArrived,
				state:        tt.fields.state,
			}
			tag.setStateAt(tt.args.newState, tt.args.timestamp)
			assert.Equal(t, tag.state, tt.args.newState)
			assert.Equal(t, tag.LastArrived, tt.args.timestamp)
			assert.Equal(t, tag.LastDeparted, tt.args.timestamp)
		})
	}
}

func TestResetStats(t *testing.T) {
	type fields struct {
		statsMap map[string]*tagStats
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "OK - nil case",
			fields: fields{statsMap: nil},
		},
		{
			name: "OK",
			fields: fields{statsMap: map[string]*tagStats{
				"test": newTagStats(),
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := &Tag{
				statsMap: tt.fields.statsMap,
			}
			tag.resetStats()
			assert.Equal(t, tag.statsMap, make(map[string]*tagStats))
		})
	}
}

func TestGetStats(t *testing.T) {
	type fields struct {
		statsMap map[string]*tagStats
	}
	type args struct {
		location string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *tagStats
	}{
		{
			name: "OK - location found",
			fields: fields{statsMap: map[string]*tagStats{
				"test": newTagStats(),
			}},
			args: args{location: "test"},
			want: newTagStats(),
		},
		{
			name: "OK - location not found",
			fields: fields{statsMap: map[string]*tagStats{
				"test-location": newTagStats(),
			}},
			args: args{location: "wrong-location"},
			want: newTagStats(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := &Tag{
				statsMap: tt.fields.statsMap,
			}
			got := tag.getStats(tt.args.location)
			assert.Equal(t, got, tt.want)
		})
	}
}
