#!/bin/bash

# Copyright (C) 2020 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

echo "Remove all EDGEX-FOUNDRY images"
$(docker images -a  | grep "edgexfoundry" | awk '{print $3}' | xargs docker rmi -f)

echo "Remove unused containers resources then remove dangling images"
docker rm -v $(docker ps -aqf status=exited)
docker rmi $(docker images -qf dangling=true)
