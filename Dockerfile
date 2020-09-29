#
# Copyright (c) 2020 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

FROM golang:1.15-alpine AS builder

# add git for go modules
RUN apk update && apk add --no-cache make git gcc libc-dev libsodium-dev zeromq-dev
WORKDIR /rfid-inventory-service

COPY go.mod .
RUN go mod download

COPY . .

# To run tests in the build container:
#   docker build --build-arg 'MAKE=build test' .
ARG MAKE='build'
RUN make $MAKE

FROM alpine

LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2020: Intel'

RUN apk --no-cache add zeromq

COPY --from=builder /rfid-inventory-service/res /res
COPY --from=builder /rfid-inventory-service/rfid-inventory /
COPY --from=builder /rfid-inventory-service/LICENSE /
COPY --from=builder /rfid-inventory-service/Attribution.txt /
ENTRYPOINT ["/rfid-inventory"]
CMD ["-cp=consul://edgex-core-consul:8500","-registry","-confdir=/res","-profile=docker"]
