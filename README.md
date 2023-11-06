# RFID LLRP Inventory Application Service
[![Build Status](https://jenkins.edgexfoundry.org/view/EdgeX%20Foundry%20Project/job/edgexfoundry/job/app-rfid-llrp-inventory/job/main/badge/icon)](https://jenkins.edgexfoundry.org/view/EdgeX%20Foundry%20Project/job/edgexfoundry/job/app-rfid-llrp-inventory/job/main/) [![Go Report Card](https://goreportcard.com/badge/github.com/edgexfoundry/app-rfid-llrp-inventory)](https://goreportcard.com/report/github.com/edgexfoundry/app-rfid-llrp-inventory) [![GitHub Latest Dev Tag)](https://img.shields.io/github/v/tag/edgexfoundry/app-rfid-llrp-inventory?include_prereleases&sort=semver&label=latest-dev)](https://github.com/edgexfoundry/app-rfid-llrp-inventory/tags) ![GitHub Latest Stable Tag)](https://img.shields.io/github/v/tag/edgexfoundry/app-rfid-llrp-inventory?sort=semver&label=latest-stable) [![GitHub License](https://img.shields.io/github/license/edgexfoundry/app-rfid-llrp-inventory)](https://choosealicense.com/licenses/apache-2.0/) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/edgexfoundry/app-rfid-llrp-inventory) [![GitHub Pull Requests](https://img.shields.io/github/issues-pr-raw/edgexfoundry/app-rfid-llrp-inventory)](https://github.com/edgexfoundry/app-rfid-llrp-inventory/pulls) [![GitHub Contributors](https://img.shields.io/github/contributors/edgexfoundry/app-rfid-llrp-inventory)](https://github.com/edgexfoundry/app-rfid-llrp-inventory/contributors) [![GitHub Committers](https://img.shields.io/badge/team-committers-green)](https://github.com/orgs/edgexfoundry/teams/app-rfid-llrp-inventory-committers/members) [![GitHub Commit Activity](https://img.shields.io/github/commit-activity/m/edgexfoundry/app-rfid-llrp-inventory)](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits)

> **Warning**  
> The **main** branch of this repository contains work-in-progress development code for the upcoming release, and is **not guaranteed to be stable or working**.
> It is only compatible with the [main branch of edgex-compose](https://github.com/edgexfoundry/edgex-compose) which uses the Docker images built from the **main** branch of this repo and other repos.
>
> **The source for the latest release can be found at [Releases](https://github.com/edgexfoundry/app-rfid-llrp-inventory/releases).**

## Overview
RFID LLRP Inventory - Edgex application service for processing tag reads,
producing events [`Arrived`, `Moved`, `Departed`], configure and manage the LLRP readers via commands

## Documentation

For latest documentation please visit https://docs.edgexfoundry.org/latest/microservices/application/services/AppLLRPInventory/Purpose

## Build Instructions

1. Clone the device-rest-go repo with the following command:

        git clone https://github.com/edgexfoundry/app-rfid-llrp-inventory.git

2. Build a docker image by using the following command:

        make docker

3. Alternatively the device service can be built natively:

        make build

## Build with NATS Messaging
Currently, the NATS Messaging capability (NATS MessageBus) is opt-in at build time.
This means that the published Docker image does not include the NATS messaging capability.

The following make commands will build the local binary or local Docker image with NATS messaging
capability included.
```makefile
make build-nats
make docker-nats
```

The locally built Docker image can then be used in place of the published Docker image in your compose file.
See [Compose Builder](https://github.com/edgexfoundry/edgex-compose/tree/main/compose-builder#gen) `nat-bus` option to generate compose file for NATS and local dev images.

## Packaging

This component is packaged as docker image.

For docker, please refer to the [Dockerfile](Dockerfile) and [Docker Compose Builder](https://github.com/edgexfoundry/edgex-compose/tree/main/compose-builder) scripts.

