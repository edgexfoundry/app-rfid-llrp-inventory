//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package behavior

import "net/http"

func d() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
	}
}
