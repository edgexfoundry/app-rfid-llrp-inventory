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

#### Build ####

```
make build
```

#### Execute unit tests with coverage ####

```
make test
```

#### Format ####

```
make fmt
```


#### Build Docker image ####
```
make docker
```

#### Docker-compose run with other Edgex services (Geneva Release) ####
```
make run
```

#### Docker-compose stop ####
```
make stop
```

#### Docker-compose down ####
```
make down
```

## License
[Apache-2.0](LICENSE)
