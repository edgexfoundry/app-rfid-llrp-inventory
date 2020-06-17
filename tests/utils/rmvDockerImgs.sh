#!/bin/bash

# Copyright (C) 2020 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Remove all EDGEX-FOUNDRY images except device-llrp-go
$(docker images -a | grep  -v "device-llrp-go" | grep "edgexfoundry" | awk '{print $3}' | xargs docker rmi -f)

# Stop/Remove unused containers resources then remove dangling images and volume
$(docker rm -v $(docker ps -aqf status=exited))
$(docker rmi $(docker images -qf dangling=true))
$(docker volume rm $(docker volume ls -qf dangling=true))