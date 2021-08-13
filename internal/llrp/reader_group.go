//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strings"
	"sync"
)

const defaultROSpecID = 1

// ROGenerator generates a new ROSpec from a Behavior and Environment,
// or returns an error if it cannot produce an ROSpec to satisfy the constraints.
type ROGenerator interface {
	NewROSpec(b Behavior, e Environment) (*ROSpec, error)
}

// ReportProcessor is anything that can accept a list of TagReportData.
type ReportProcessor interface {
	ProcessTagReport(tags []TagReportData)
}

// TagReader is something which can process TagReportData
// generated as a result of executing any ROSpec it generates.
//
// It can and should assume that TagReportData resulted from an ROSpec it generated,
// though whether that ROSpec is the most recent it generated
// is up to (and should be specified by) the particular TagReader implementation.
type TagReader interface {
	ROGenerator
	ReportProcessor
}

// A ReaderGroup unites a collection of named TagReader instances
// with a single, specific Behavior and Environment.
type ReaderGroup struct {
	mu       sync.RWMutex
	readers  map[string]TagReader
	env      Environment
	behavior Behavior
}

func NewReaderGroup() *ReaderGroup {
	return &ReaderGroup{
		readers: map[string]TagReader{},
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

// Behavior returns a (shallow) copy of the ReaderGroup's current Behavior.
func (rg *ReaderGroup) Behavior() Behavior {
	rg.mu.RLock()
	b := rg.behavior
	rg.mu.RUnlock()
	return b
}

// WriteReaders writes to w a JSON-formatted list of readers in this group.
func (rg *ReaderGroup) WriteReaders(w io.Writer) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	s := struct{ Readers []string }{Readers: make([]string, 0, len(rg.readers))}
	for r := range rg.readers {
		s.Readers = append(s.Readers, r)
	}

	return json.NewEncoder(w).Encode(s)
}

// ProcessTagReport uses the named TagReader
// to process the list of TagReportData.
//
// If no Reader in the ReaderGroup matches the given name,
// this method returns false.
// Otherwise, it returns true.
func (rg *ReaderGroup) ProcessTagReport(name string, tags []TagReportData) bool {
	rg.mu.RLock()
	tr, ok := rg.readers[name]
	rg.mu.RUnlock()

	if !ok {
		return false
	}

	tr.ProcessTagReport(tags)
	return true
}

// RemoveReader removes the named Reader from the ReaderGroup, if present.
// If no Reader with that name is in the ReaderGroup, nothing happens.
func (rg *ReaderGroup) RemoveReader(name string) {
	rg.mu.Lock()
	delete(rg.readers, name)
	rg.mu.Unlock()
}

// AddReader asks the ReaderGroup to manage a TagReader with given name.
//
// First, it uses the name to request a TagReader from the DSClient,
// then it uses that TagReader to generate an ROSpec
// based on the ReaderGroup's Behavior and Environment.
// Finally, it uses the DSClient to replace that device's ROSpec with the new one.
//
// If these steps all succeed, the ReaderGroup accepts the TagReader,
// possibly replacing a previously-held TagReader with the same name.
// Any changes to the ReaderGroup's Behavior must be accepted by the TagReader,
// and ReaderGroup will send appropriate commands to the TagReader
// in response to calls to StartAll or StopAll.
//
// On failure, the ReaderGroup rejects the TagReader and returns an error.
// Because part of this process attempts to replace the device's ROSpec,
// it's possible that device's ROSpec is deleted without a new one replacing it.
func (rg *ReaderGroup) AddReader(ds DSClient, name string) error {
	r, err := ds.NewReader(name)
	if err != nil {
		return err
	}

	rg.mu.RLock()
	env := rg.env
	b := rg.behavior
	rg.mu.RUnlock()

	s, err := r.NewROSpec(b, env)
	if err != nil {
		return err
	}

	s.ROSpecID = defaultROSpecID
	if err := replaceRO(ds, name, s); err != nil {
		return err
	}

	rg.mu.Lock()
	rg.readers[name] = r
	rg.mu.Unlock()

	ds.lc.Info(fmt.Sprintf("Successfully added device %s to default group.", name))

	return nil
}

// replaceRO deletes any ROSpec on the named device, then adds the given ROSpec.
// This won't try to Add the ROSpec unless the delete is successful,
// but it's possible the delete succeeds but the add fails.
func replaceRO(ds DSClient, name string, spec *ROSpec) error {
	if err := ds.DeleteAllROSpecs(name); err != nil {
		return err
	}

	return ds.AddROSpec(name, spec)
}

// SetBehavior changes the ReaderGroup's Behavior.
//
// The new Behavior must be valid for every TagReader in the ReaderGroup.
// Before accepting it, this method uses the new Behavior and current Environment
// to generate new ROSpecs for each TagReader in the ReaderGroup.
// If any TagReader rejects the Behavior,
// the ReaderGroup rejects the Behavior and returns an error,
// leaving it with the Behavior it had before it was called.
//
// If every TagReader in the ReaderGroup can implement the Behavior,
// the ReaderGroup accepts the new Behavior,
// will use it when evaluating whether to accept a new TagReader,
// and will return it from calls to ReaderGroup.Behavior().
//
// Before this method returns, assuming the Behavior is accepted,
// it concurrently sends each newly generated ROSpec to the appropriate TagReader.
// Any errors returned by this step are collected into a MultiErr
// which is returned after the last update call completes.
// A failure to set one TagReader's ROSpec does not have an impact on others.
// The ReaderGroup has still accepted the new Behavior
// and will continue to use it in future calls.
//
// It is safe to call SetBehavior multiple times with the same Behavior,
// although doing so will reapply it to every TagReader in the ReaderGroup.
func (rg *ReaderGroup) SetBehavior(ds DSClient, b Behavior) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	specs := map[string]*ROSpec{}
	for name, r := range rg.readers {
		s, err := r.NewROSpec(b, rg.env)
		if err != nil {
			return errors.WithMessagef(err, "new behavior is invalid for %q", name)
		}

		s.ROSpecID = defaultROSpecID
		specs[name] = s
	}

	// The behavior is valid for all members of the group.
	rg.behavior = b

	// Replace each reader's ROSpec.
	errs := make(chan error, len(specs))
	wg := sync.WaitGroup{}
	wg.Add(len(specs))
	for d, s := range specs {
		go func(name string, s *ROSpec) {
			defer wg.Done()
			if err := replaceRO(ds, name, s); err != nil {
				errs <- errors.WithMessagef(err, "failed to replace ROSpec for %q", name)
			}
		}(d, s)
	}

	// Wait for the replace calls to complete, then collect any errors.
	wg.Wait()
	close(errs)
	var multiErr MultiErr
	for err := range errs {
		multiErr = append(multiErr, err)
	}

	if len(multiErr) == 0 {
		return nil
	}

	return errors.WithMessagef(multiErr,
		"failed to replace ROSpec on %d readers", len(multiErr))
}

// MultiErr tracks a list of errors collected
// when an operation is applied to multiple things.
type MultiErr []error

// Error implements the error interface for MultiErr
// by returning a single string listing all the collected errors,
// separated by a semicolon and a space ("; ").
func (me MultiErr) Error() string {
	strs := make([]string, len(me))
	for i, s := range me {
		strs[i] = s.Error()
	}

	return strings.Join(strs, "; ")
}

// StartAll uses the DSClient to start all TagReaders in the ReaderGroup.
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

// StopAll uses the DSClient to stop all TagReaders in the ReaderGroup.
func (rg *ReaderGroup) StopAll(ds DSClient) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	var errs []error
	for name := range rg.readers {
		if rg.behavior.StartTrigger().Trigger == ROStartTriggerNone {
			if err := ds.StopROSpec(name, 1); err != nil {
				errs = append(errs, err)
			}
		}

		if err := ds.DisableROSpec(name, 1); err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return MultiErr(errs)
	}
	return nil
}
