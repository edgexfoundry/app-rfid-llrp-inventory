// Code generated by go-swagger; DO NOT EDIT.

//
// Copyright NetFoundry Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// __          __              _
// \ \        / /             (_)
//  \ \  /\  / /_ _ _ __ _ __  _ _ __   __ _
//   \ \/  \/ / _` | '__| '_ \| | '_ \ / _` |
//    \  /\  / (_| | |  | | | | | | | | (_| | : This file is generated, do not edit it.
//     \/  \/ \__,_|_|  |_| |_|_|_| |_|\__, |
//                                      __/ |
//                                     |___/

package informational

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewListVersionParams creates a new ListVersionParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListVersionParams() *ListVersionParams {
	return &ListVersionParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListVersionParamsWithTimeout creates a new ListVersionParams object
// with the ability to set a timeout on a request.
func NewListVersionParamsWithTimeout(timeout time.Duration) *ListVersionParams {
	return &ListVersionParams{
		timeout: timeout,
	}
}

// NewListVersionParamsWithContext creates a new ListVersionParams object
// with the ability to set a context for a request.
func NewListVersionParamsWithContext(ctx context.Context) *ListVersionParams {
	return &ListVersionParams{
		Context: ctx,
	}
}

// NewListVersionParamsWithHTTPClient creates a new ListVersionParams object
// with the ability to set a custom HTTPClient for a request.
func NewListVersionParamsWithHTTPClient(client *http.Client) *ListVersionParams {
	return &ListVersionParams{
		HTTPClient: client,
	}
}

/* ListVersionParams contains all the parameters to send to the API endpoint
   for the list version operation.

   Typically these are written to a http.Request.
*/
type ListVersionParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list version params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVersionParams) WithDefaults() *ListVersionParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list version params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVersionParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list version params
func (o *ListVersionParams) WithTimeout(timeout time.Duration) *ListVersionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list version params
func (o *ListVersionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list version params
func (o *ListVersionParams) WithContext(ctx context.Context) *ListVersionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list version params
func (o *ListVersionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list version params
func (o *ListVersionParams) WithHTTPClient(client *http.Client) *ListVersionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list version params
func (o *ListVersionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListVersionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
