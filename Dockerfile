#
# Copyright (c) 2023 Intel Corporation
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

# build stage
ARG BASE=golang:1.18-alpine3.16
FROM ${BASE} AS builder

ARG ALPINE_PKG_BASE="make git"
ARG ALPINE_PKG_EXTRA=""
ARG ADD_BUILD_TAGS=""

RUN apk add --no-cache ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}
WORKDIR /app

COPY go.mod vendor* ./
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

COPY . .
ARG MAKE="make -e ADD_BUILD_TAGS=$ADD_BUILD_TAGS build"
RUN $MAKE

# final stage
FROM alpine:3.14
LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2023: Intel'
LABEL Name=app-service-rfid-llrp-inventory Version=${VERSION}

RUN apk --no-cache add ca-certificates dumb-init

COPY --from=builder /app/Attribution.txt /Attribution.txt
COPY --from=builder /app/LICENSE /LICENSE
COPY --from=builder /app/res/ /res/
COPY --from=builder /app/static/ /static/
COPY --from=builder /app/app-rfid-llrp-inventory /app-rfid-llrp-inventory

RUN mkdir -p /cache && \
    chown -R 2002:2002 /cache

VOLUME /cache

EXPOSE 59711

ENTRYPOINT ["/app-rfid-llrp-inventory"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "-registry"]
