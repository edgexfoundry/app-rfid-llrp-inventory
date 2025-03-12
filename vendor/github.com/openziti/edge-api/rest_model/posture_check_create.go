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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// PostureCheckCreate posture check create
//
// swagger:discriminator postureCheckCreate typeId
type PostureCheckCreate interface {
	runtime.Validatable
	runtime.ContextValidatable

	// name
	// Required: true
	Name() *string
	SetName(*string)

	// role attributes
	RoleAttributes() *Attributes
	SetRoleAttributes(*Attributes)

	// tags
	Tags() *Tags
	SetTags(*Tags)

	// type Id
	// Required: true
	TypeID() PostureCheckType
	SetTypeID(PostureCheckType)

	// AdditionalProperties in base type shoud be handled just like regular properties
	// At this moment, the base type property is pushed down to the subtype
}

type postureCheckCreate struct {
	nameField *string

	roleAttributesField *Attributes

	tagsField *Tags

	typeIdField PostureCheckType
}

// Name gets the name of this polymorphic type
func (m *postureCheckCreate) Name() *string {
	return m.nameField
}

// SetName sets the name of this polymorphic type
func (m *postureCheckCreate) SetName(val *string) {
	m.nameField = val
}

// RoleAttributes gets the role attributes of this polymorphic type
func (m *postureCheckCreate) RoleAttributes() *Attributes {
	return m.roleAttributesField
}

// SetRoleAttributes sets the role attributes of this polymorphic type
func (m *postureCheckCreate) SetRoleAttributes(val *Attributes) {
	m.roleAttributesField = val
}

// Tags gets the tags of this polymorphic type
func (m *postureCheckCreate) Tags() *Tags {
	return m.tagsField
}

// SetTags sets the tags of this polymorphic type
func (m *postureCheckCreate) SetTags(val *Tags) {
	m.tagsField = val
}

// TypeID gets the type Id of this polymorphic type
func (m *postureCheckCreate) TypeID() PostureCheckType {
	return "postureCheckCreate"
}

// SetTypeID sets the type Id of this polymorphic type
func (m *postureCheckCreate) SetTypeID(val PostureCheckType) {
}

// UnmarshalPostureCheckCreateSlice unmarshals polymorphic slices of PostureCheckCreate
func UnmarshalPostureCheckCreateSlice(reader io.Reader, consumer runtime.Consumer) ([]PostureCheckCreate, error) {
	var elements []json.RawMessage
	if err := consumer.Consume(reader, &elements); err != nil {
		return nil, err
	}

	var result []PostureCheckCreate
	for _, element := range elements {
		obj, err := unmarshalPostureCheckCreate(element, consumer)
		if err != nil {
			return nil, err
		}
		result = append(result, obj)
	}
	return result, nil
}

// UnmarshalPostureCheckCreate unmarshals polymorphic PostureCheckCreate
func UnmarshalPostureCheckCreate(reader io.Reader, consumer runtime.Consumer) (PostureCheckCreate, error) {
	// we need to read this twice, so first into a buffer
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return unmarshalPostureCheckCreate(data, consumer)
}

func unmarshalPostureCheckCreate(data []byte, consumer runtime.Consumer) (PostureCheckCreate, error) {
	buf := bytes.NewBuffer(data)
	buf2 := bytes.NewBuffer(data)

	// the first time this is read is to fetch the value of the typeId property.
	var getType struct {
		TypeID string `json:"typeId"`
	}
	if err := consumer.Consume(buf, &getType); err != nil {
		return nil, err
	}

	if err := validate.RequiredString("typeId", "body", getType.TypeID); err != nil {
		return nil, err
	}

	// The value of typeId is used to determine which type to create and unmarshal the data into
	switch getType.TypeID {
	case "DOMAIN":
		var result PostureCheckDomainCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	case "MAC":
		var result PostureCheckMacAddressCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	case "MFA":
		var result PostureCheckMfaCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	case "OS":
		var result PostureCheckOperatingSystemCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	case "PROCESS":
		var result PostureCheckProcessCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	case "PROCESS_MULTI":
		var result PostureCheckProcessMultiCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	case "postureCheckCreate":
		var result postureCheckCreate
		if err := consumer.Consume(buf2, &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, errors.New(422, "invalid typeId value: %q", getType.TypeID)
}

// Validate validates this posture check create
func (m *postureCheckCreate) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRoleAttributes(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTags(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *postureCheckCreate) validateName(formats strfmt.Registry) error {

	if err := validate.Required("name", "body", m.Name()); err != nil {
		return err
	}

	return nil
}

func (m *postureCheckCreate) validateRoleAttributes(formats strfmt.Registry) error {
	if swag.IsZero(m.RoleAttributes()) { // not required
		return nil
	}

	if m.RoleAttributes() != nil {
		if err := m.RoleAttributes().Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("roleAttributes")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("roleAttributes")
			}
			return err
		}
	}

	return nil
}

func (m *postureCheckCreate) validateTags(formats strfmt.Registry) error {
	if swag.IsZero(m.Tags()) { // not required
		return nil
	}

	if m.Tags() != nil {
		if err := m.Tags().Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("tags")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("tags")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this posture check create based on the context it is used
func (m *postureCheckCreate) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateRoleAttributes(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateTags(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateTypeID(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *postureCheckCreate) contextValidateRoleAttributes(ctx context.Context, formats strfmt.Registry) error {

	if m.RoleAttributes() != nil {
		if err := m.RoleAttributes().ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("roleAttributes")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("roleAttributes")
			}
			return err
		}
	}

	return nil
}

func (m *postureCheckCreate) contextValidateTags(ctx context.Context, formats strfmt.Registry) error {

	if m.Tags() != nil {
		if err := m.Tags().ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("tags")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("tags")
			}
			return err
		}
	}

	return nil
}

func (m *postureCheckCreate) contextValidateTypeID(ctx context.Context, formats strfmt.Registry) error {

	if err := m.TypeID().ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("typeId")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("typeId")
		}
		return err
	}

	return nil
}
