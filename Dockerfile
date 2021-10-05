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

# build stage
ARG BASE=golang:1.16-alpine3.14
FROM ${BASE} AS builder

ARG ALPINE_PKG_BASE="make git gcc libc-dev libsodium-dev zeromq-dev"
ARG ALPINE_PKG_EXTRA=""

LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2020: Intel'
RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
RUN apk add --no-cache ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}
WORKDIR /app

COPY . .
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

ARG MAKE="make build"
RUN $MAKE

# final stage
FROM alpine:3.14
LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2020: Intel'
LABEL Name=app-service-rfid-llrp-inventory Version=${VERSION}

RUN apk --no-cache add ca-certificates zeromq

COPY --from=builder /app/Attribution.txt /Attribution.txt
COPY --from=builder /app/LICENSE /LICENSE
COPY --from=builder /app/res/ /res/
COPY --from=builder /app/static/ /static/
COPY --from=builder /app/app-rfid-llrp-inventory /app-rfid-llrp-inventory

EXPOSE 48086

ENTRYPOINT ["/app-rfid-llrp-inventory"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "-registry", "-confdir=/res"]
