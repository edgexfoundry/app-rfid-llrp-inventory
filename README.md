# RFID Inventory Service
## Overview
RFID Inventory Service - Edgex application service for processing tag reads,
producing events [`Arrived`, `Moved`, `Departed`], configure and manage the LLRP readers via commands

## Installation and Execution ##

#### Prerequisites ####

 - Go language
 - GNU Make
 - Docker
 - Docker-compose

##### Build #####
```
$ make build
```
##### Execute unit tests with coverage #####
```
$ make test
```
##### Format #####
```
$ make fmt
```
##### Build Docker image #####
```
$ make docker
```

#### Commands Available
- Ping command to see if the service is up and running.
```
GET http://localhost:48086/ping

pong
```
- Command to get all the list of LLRP readers registered in edgex.
```
GET http://localhost:48086/command/readers

{"ReaderList":["192.168.1.78_5084"]}
```
- Command to make the LLRP reader start reading tags
```
POST http://localhost:48086/command/readings/StartReading

OK
```
- Command to make the LLRP reader stop reading tags
```
POST http://localhost:48086/command/readings/StopReading

OK
```

## Inventory Events
There are 3 basic inventory events that are generated and sent to EdgeX's core-data. 
Here are some example `EdgeX Readings`.

- **`InventoryEventArrived`**
```json
{
  "id": "6def8859-5a12-4c83-b68c-256303146682",
  "device": "rfid-inventory",
  "created": 1598043284110,
  "origin": 1598043284109799400,
  "readings": [
    {
      "id": "8d15d035-402f-4abc-85fc-a7ed7213122a",
      "created": 1598043284110,
      "origin": 1598043284109799400,
      "device": "rfid-inventory",
      "name": "InventoryEventArrived",
      "value": "{\"epc\":\"30340bb6884cb101a13bc744\",\"timestamp\":1598043284104,\"location\":\"SpeedwayR-10-EF-18_1\"}"
    }
  ]
}
```

- **`InventoryEventMoved`**
```json
{
  "id": "c78c304e-1906-4d17-bf26-5075756a231f",
  "device": "rfid-inventory",
  "created": 1598401259699,
  "origin": 1598401259697580500,
  "readings": [
    {
      "id": "323694d9-1a48-417a-9f43-25998536ae8f",
      "created": 1598401259699,
      "origin": 1598401259697580500,
      "device": "rfid-inventory",
      "name": "InventoryEventMoved",
      "value": "{\"epc\":\"30340bb6884cb101a13bc744\",\"timestamp\":1598401259691,\"prev_location\":\"SpeedwayR-10-EF-18_1\",\"location\":\"SpeedwayR-10-EF-18_3\"}"
    }
  ]
}
```

- **`InventoryEventDeparted`**
```json
{
  "id": "4d042708-c5de-41fa-827a-3f24b364c6de",
  "device": "rfid-inventory",
  "created": 1598062424895,
  "origin": 1598062424894043600,
  "readings": [
    {
      "id": "928ff90d-02d1-43be-81a6-a0d75886b0e4",
      "created": 1598062424895,
      "origin": 1598062424894043600,
      "device": "rfid-inventory",
      "name": "InventoryEventDeparted",
      "value": "{\"epc\":\"30340bb6884cb101a13bc744\",\"timestamp\":1598062424893,\"last_read\":1598062392524,\"last_location\":\"SpeedwayR-10-EF-18_1\"}"
    }
  ]
}
```


### Arrived
Arrived events are generated when _**ANY**_ of the following conditions are met:
- A tag is read that has never been read before
- A tag is read that is currently in the Departed state
- A tag aged-out of the inventory and has been read again

### Moved
Moved events are generated when _**ALL**_ of the following conditions are met:
- A tag is read by an Antenna (`Incoming Antenna`) that is not the current Location
- The `Incoming Antenna`'s Alias does not match the current Location's Alias
- The `Incoming Antenna` has read that tag at least `2` times total (including this one)
- The moving average of RSSI values from the `Incoming Antenna` are greater than the 
  current Location's _**weighted**_ moving average _([See: Mobility Profile](#Mobility-profile))_

### Departed
Departed events are generated when:
- A tag is in the `Present` state and has not been read in more than 
  the configured `DepartedThresholdSeconds`

_NOTE: Departed tags have their tag statistics cleared, essentially resetting any values used
       by the tag algorithm. So if this tag is seen again, the Location will be set to the
       first Antenna that reads the tag again._

### Tag State Machine
Here is a diagram of the internal tag state machine. Every tag starts in the `Unknown` state (more precisely does not exist at all in memory). 
Throughout the lifecycle of the tag, events will be generated that will cause it to move between
`Present` and `Departed`. Eventually once a tag has been in the `Departed` state for long enough
it will "Age Out" which removes it from memory, effectively putting it back into the `Unknown` state.

![Tag State Diagram](docs/images/tag-state-diagram.png)

## Tag Location Algorithm

Every tag is associated with a single `Location` which is the best estimation of the Reader and Antenna
that this tag is closest to. 

**Note:** _Locations are actually based on `Aliases` and multiple antennas may be mapped to the 
same `Alias`, which will cause them to be treated as the same within the tag algorithm. This can be
especially useful when using a dual-linear antenna and mapping both polarities to the same `Alias`._


### Configuration

The following configuration options affect how the tag location algorithm works under the hood.

- **`TagStatsWindowSize`** *`[int]`*: How many reads to keep track of *per alias* for each RFID tag. 
        This effects how many tag reads will be used when computing the rolling average for tag stats.
  - default: `20`

- **`AdjustLastReadOnByOrigin`** *`[bool]`*: If `true`, this will override the tag read timestamps sent from the sensor
        with an adjusted one based on the UTC time the `LLRP Device Service` received the message from the device (aka `Origin`). 
        Essentially all timestamps will be shifted by the difference in time from when the sensor says it was read versus when it
        was actually received. This option attempts to account for message latency and time drift between sensors by standardizing 
        all timestamps. If `false`, timestamps will retain their original values sent from the sensor.
  - default: `true`
  - computation: `readOn = (Origin - sentOn) + readOn`

- **`DepartedThresholdSeconds`** *`[int]`*: How long in seconds a tag should not be read before 
        it will generate a `Departed` event.
  - default: `30`

- **`DepartedCheckIntervalSeconds`** *`[int]`*: How often to run the background task that checks if a Tag needs
        to be marked `Departed`. Smaller intervals will cause more frequent checks and less variability at the expense of
        CPU utilization and lock contention. Larger intervals on the other hand may cause greater latency
        between when a tag passes the `DepartedThresholdSeconds` and when the `Departed` event is actually
        generated (waiting for the next check to occur).
  - default: `10`
  
- **`AgeOutHours`** *`[int]`*: How long in hours to keep `Departed` tags in our in-memory inventory before they 
        are aged-out (purged). This is done for CPU and RAM conservation in deployments with a large
        turnover of unique tags. An aged-out tag will be purged from memory and if it is 
        read again it will be treated as the first time seeing that tag.
  - default: `336` _(aka: 2 weeks)_

### Mobility Profile

The following configuration options define the `Mobility Profile` values.
These values are used in the Location algorithm as a weighting function which
will decay RSSI values over time. This weight is then applied to the existing Tag's Location
and compared to the non-weighted average.

Suppose the following variables:
- **`incomingRSSI`** Mean RSSI of last `windowSize` reads by incoming read's location 
- **`existingRSSI`** Mean RSSI of last `windowSize` reads by tag's existing location
- **`weight`** Result of Mobility Profile's computations

The location will change when the following equation is true:
- `incomingRSSI > (existingRSSI * weight)`

![Mobility Profile Diagram](docs/images/mobility-profile.png)

- **`MobilityProfileBaseProfile`** *`[enum]`*: Name of the parent mobility profile to inherit from. Any values which are not explicitly overridden will be inherited from this base profile selected.
  - default: `'default'` *(which is currently the same as `'asset_tracking'`)*
  - available options: `'default'`, `'asset_tracking'`, `'retail_garment'`

- **`MobilityProfileSlope`** *`[float]`*: Used to determine the weight applied to older RSSI values (aka rate of decay)
  - default: *(none, inherit from base profile)*
  - units: `dBm per millisecond`

- **`MobilityProfileThreshold`** *`[float]`*: RSSI threshold that must be exceeded for the tag to move from the previous sensor
  - default: *(none, inherit from base profile)*
  - units: `dBm`

- **`MobilityProfileHoldoffMillis`** *`[float]`*: Amount of time in which the weight used is just the threshold, effectively the slope is not used
  - default: *(none, inherit from base profile)*
  - units: `milliseconds`

## License
[Apache-2.0](LICENSE)
