//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"strings"
	"sync"
)

type TProc interface {
	NewROSpec(b Behavior, e Environment) (*ROSpec, error)
	FillAmbiguousNil(tags []TagReportData)
}

type TagReader struct {
	name string
	TProc
}

type ReaderGroup struct {
	mu       sync.RWMutex
	readers  map[string]*TagReader
	env      Environment
	behavior Behavior
}

func NewReaderGroup() *ReaderGroup {
	return &ReaderGroup{
		readers: map[string]*TagReader{},
		env:     Environment{},
		behavior: Behavior{
			ImpinjOptions: &ImpinjOptions{SuppressMonza: false},
			ScanType:      ScanNormal,
			Duration:      0,                      // infinite
			Power:         PowerTarget{Max: 3000}, // 30 dBm
			GPITrigger:    nil,
			Frequencies:   nil, // assume power is valid at all frequencies (for non-Hopping)
		},
	}
}

func (rg *ReaderGroup) Behavior() Behavior {
	return rg.behavior
}

// ListReaders writes to w a JSON-formatted list of readers in this group.
func (rg *ReaderGroup) ListReaders(w io.Writer) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	s := struct{ Readers []string }{Readers: make([]string, 0, len(rg.readers))}
	for r := range rg.readers {
		s.Readers = append(s.Readers, r)
	}

	return json.NewEncoder(w).Encode(s)
}

// IsDeepScan returns whether or not this reader group is currently performing a deep scan
// operation which can be used to adjust timeouts and algorithm values
func (rg *ReaderGroup) IsDeepScan() bool {
	return rg.behavior.ScanType == ScanDeep
}

func (rg *ReaderGroup) ProcessTagReport(name string, tags []TagReportData) bool {
	rg.mu.RLock()
	tr, ok := rg.readers[name]
	rg.mu.RUnlock()

	if !ok {
		return false
	}

	tr.FillAmbiguousNil(tags)
	return true
}

func (rg *ReaderGroup) RemoveReader(name string) {
	rg.mu.Lock()
	delete(rg.readers, name)
	rg.mu.Unlock()
}

func (rg *ReaderGroup) AddReader(ds DSClient, name string) error {
	r, err := ds.NewReader(name)
	if err != nil {
		return err
	}

	s, err := r.NewROSpec(rg.behavior, rg.env)
	if err != nil {
		return err
	}

	if err := r.replaceRO(ds, s); err != nil {
		return err
	}

	rg.mu.Lock()
	rg.readers[name] = r
	rg.mu.Unlock()

	return nil
}

func (r *TagReader) replaceRO(ds DSClient, spec *ROSpec) error {
	if err := ds.DeleteAllROSpecs(r.name); err != nil {
		return err
	}

	return ds.AddROSpec(r.name, spec)
}

// SetBehavior updates the behavior for each TagReader in the ReaderGroup.
func (rg *ReaderGroup) SetBehavior(ds DSClient, b Behavior) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	specs := map[string]*ROSpec{}
	for name, r := range rg.readers {
		s, err := r.NewROSpec(b, rg.env)
		if err != nil {
			return errors.WithMessagef(err, "new behavior is invalid for %q", name)
		}
		specs[name] = s
	}

	// the behavior is valid for all members of the group
	rg.behavior = b

	errs := make(chan error, len(specs))
	wg := sync.WaitGroup{}
	wg.Add(len(specs))
	for d, s := range specs {
		go func(name string, s *ROSpec) {
			defer wg.Done()
			if err := rg.readers[name].replaceRO(ds, s); err != nil {
				errs <- errors.WithMessagef(err, "failed to replace ROSpec for %q", name)
			}
		}(d, s)
	}

	wg.Wait()
	close(errs)
	var errStrs []string
	for err := range errs {
		if err == nil {
			continue
		}

		errStrs = append(errStrs, err.Error())
	}

	errStr := strings.Join(errStrs, "; ")
	if errStr != "" {
		return errors.Errorf("failed to set some behaviors: %s", errStr)
	}

	return nil
}

type MultiErr []error

func (me MultiErr) Error() string {
	strs := make([]string, len(me))
	for i, s := range me {
		strs[i] = s.Error()
	}

	return strings.Join(strs, "; ")
}

func (rg *ReaderGroup) StartAll(ds DSClient) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	var errs []error
	for name := range rg.readers {
		if err := ds.EnableROSpec(name, 1); err != nil {
			errs = append(errs, err)
		}

		if rg.behavior.StartTrigger().Trigger == ROStartTriggerNone {
			if err := ds.StartROSpec(name, 1); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if errs != nil {
		return MultiErr(errs)
	}
	return nil
}

func (rg *ReaderGroup) StopAll(ds DSClient) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	var errs []error
	for name := range rg.readers {
		if err := ds.DisableROSpec(name, 1); err != nil {
			errs = append(errs, err)
		}

		if rg.behavior.StartTrigger().Trigger == ROStartTriggerNone {
			if err := ds.StopROSpec(name, 1); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if errs != nil {
		return MultiErr(errs)
	}
	return nil
}
