/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package jsonrpc

import (
	"encoding/json"
	"github.com/pkg/errors"
	"strings"
)

func Decode(value string, js Message) error {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()

	if err := decoder.Decode(js); err != nil {
		return errors.Wrap(err, "error decoding jsonrpc messaage")
	}

	if err := js.Validate(); err != nil {
		return errors.Wrap(err, "error validating jsonrpc messaage")
	}

	return nil
}
