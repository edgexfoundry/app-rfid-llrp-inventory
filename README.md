# RFID Inventory Service
## Overview
RFID Inventory Service - Edgex application service for processing tag reads
and producing events [ARRIVED, MOVED]

## Installation and Execution ##

#### Prerequisites ####

 - Go language
 - GNU Make
 - Docker
 - Docker-compose
 - Ubuntu install libsodium-dev libzmq3-dev

#### Build ####

```
make build
```

#### Build Docker image ####
```
sudo make docker
```
NOTE!! The docker-compose file does not include the inventory service yet.
to support development, docker compose starts all the supporting edgex
containers, and the inventory service can be run from the IDE or command line.

#### Docker-compose run with other Edgex services (Geneva Release) ####
```
sudo make run
```

#### Docker-compose stop ####
```
sudo make stop
```
## License
[Apache-2.0](LICENSE)
