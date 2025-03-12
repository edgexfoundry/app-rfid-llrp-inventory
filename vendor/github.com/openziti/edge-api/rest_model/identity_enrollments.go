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

package rest_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// IdentityEnrollments identity enrollments
//
// swagger:model identityEnrollments
type IdentityEnrollments struct {

	// ott
	Ott *IdentityEnrollmentsOtt `json:"ott,omitempty"`

	// ottca
	Ottca *IdentityEnrollmentsOttca `json:"ottca,omitempty"`

	// updb
	Updb *IdentityEnrollmentsUpdb `json:"updb,omitempty"`
}

// Validate validates this identity enrollments
func (m *IdentityEnrollments) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateOtt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOttca(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUpdb(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IdentityEnrollments) validateOtt(formats strfmt.Registry) error {
	if swag.IsZero(m.Ott) { // not required
		return nil
	}

	if m.Ott != nil {
		if err := m.Ott.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ott")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("ott")
			}
			return err
		}
	}

	return nil
}

func (m *IdentityEnrollments) validateOttca(formats strfmt.Registry) error {
	if swag.IsZero(m.Ottca) { // not required
		return nil
	}

	if m.Ottca != nil {
		if err := m.Ottca.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ottca")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("ottca")
			}
			return err
		}
	}

	return nil
}

func (m *IdentityEnrollments) validateUpdb(formats strfmt.Registry) error {
	if swag.IsZero(m.Updb) { // not required
		return nil
	}

	if m.Updb != nil {
		if err := m.Updb.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("updb")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("updb")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this identity enrollments based on the context it is used
func (m *IdentityEnrollments) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateOtt(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOttca(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateUpdb(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IdentityEnrollments) contextValidateOtt(ctx context.Context, formats strfmt.Registry) error {

	if m.Ott != nil {
		if err := m.Ott.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ott")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("ott")
			}
			return err
		}
	}

	return nil
}

func (m *IdentityEnrollments) contextValidateOttca(ctx context.Context, formats strfmt.Registry) error {

	if m.Ottca != nil {
		if err := m.Ottca.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ottca")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("ottca")
			}
			return err
		}
	}

	return nil
}

func (m *IdentityEnrollments) contextValidateUpdb(ctx context.Context, formats strfmt.Registry) error {

	if m.Updb != nil {
		if err := m.Updb.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("updb")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("updb")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *IdentityEnrollments) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IdentityEnrollments) UnmarshalBinary(b []byte) error {
	var res IdentityEnrollments
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// IdentityEnrollmentsOtt identity enrollments ott
//
// swagger:model IdentityEnrollmentsOtt
type IdentityEnrollmentsOtt struct {

	// expires at
	// Format: date-time
	ExpiresAt strfmt.DateTime `json:"expiresAt,omitempty"`

	// id
	ID string `json:"id,omitempty"`

	// jwt
	JWT string `json:"jwt,omitempty"`

	// token
	Token string `json:"token,omitempty"`
}

// Validate validates this identity enrollments ott
func (m *IdentityEnrollmentsOtt) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateExpiresAt(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IdentityEnrollmentsOtt) validateExpiresAt(formats strfmt.Registry) error {
	if swag.IsZero(m.ExpiresAt) { // not required
		return nil
	}

	if err := validate.FormatOf("ott"+"."+"expiresAt", "body", "date-time", m.ExpiresAt.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this identity enrollments ott based on context it is used
func (m *IdentityEnrollmentsOtt) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IdentityEnrollmentsOtt) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IdentityEnrollmentsOtt) UnmarshalBinary(b []byte) error {
	var res IdentityEnrollmentsOtt
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// IdentityEnrollmentsOttca identity enrollments ottca
//
// swagger:model IdentityEnrollmentsOttca
type IdentityEnrollmentsOttca struct {

	// ca
	Ca *EntityRef `json:"ca,omitempty"`

	// ca Id
	CaID string `json:"caId,omitempty"`

	// expires at
	// Format: date-time
	ExpiresAt strfmt.DateTime `json:"expiresAt,omitempty"`

	// id
	ID string `json:"id,omitempty"`

	// jwt
	JWT string `json:"jwt,omitempty"`

	// token
	Token string `json:"token,omitempty"`
}

// Validate validates this identity enrollments ottca
func (m *IdentityEnrollmentsOttca) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCa(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateExpiresAt(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IdentityEnrollmentsOttca) validateCa(formats strfmt.Registry) error {
	if swag.IsZero(m.Ca) { // not required
		return nil
	}

	if m.Ca != nil {
		if err := m.Ca.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ottca" + "." + "ca")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("ottca" + "." + "ca")
			}
			return err
		}
	}

	return nil
}

func (m *IdentityEnrollmentsOttca) validateExpiresAt(formats strfmt.Registry) error {
	if swag.IsZero(m.ExpiresAt) { // not required
		return nil
	}

	if err := validate.FormatOf("ottca"+"."+"expiresAt", "body", "date-time", m.ExpiresAt.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validate this identity enrollments ottca based on the context it is used
func (m *IdentityEnrollmentsOttca) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCa(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IdentityEnrollmentsOttca) contextValidateCa(ctx context.Context, formats strfmt.Registry) error {

	if m.Ca != nil {
		if err := m.Ca.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ottca" + "." + "ca")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("ottca" + "." + "ca")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *IdentityEnrollmentsOttca) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IdentityEnrollmentsOttca) UnmarshalBinary(b []byte) error {
	var res IdentityEnrollmentsOttca
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// IdentityEnrollmentsUpdb identity enrollments updb
//
// swagger:model IdentityEnrollmentsUpdb
type IdentityEnrollmentsUpdb struct {

	// expires at
	// Format: date-time
	ExpiresAt strfmt.DateTime `json:"expiresAt,omitempty"`

	// id
	ID string `json:"id,omitempty"`

	// jwt
	JWT string `json:"jwt,omitempty"`

	// token
	Token string `json:"token,omitempty"`
}

// Validate validates this identity enrollments updb
func (m *IdentityEnrollmentsUpdb) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateExpiresAt(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IdentityEnrollmentsUpdb) validateExpiresAt(formats strfmt.Registry) error {
	if swag.IsZero(m.ExpiresAt) { // not required
		return nil
	}

	if err := validate.FormatOf("updb"+"."+"expiresAt", "body", "date-time", m.ExpiresAt.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this identity enrollments updb based on context it is used
func (m *IdentityEnrollmentsUpdb) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IdentityEnrollmentsUpdb) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IdentityEnrollmentsUpdb) UnmarshalBinary(b []byte) error {
	var res IdentityEnrollmentsUpdb
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
