# EdgeX Foundry RFID-LLRP Inventory App Service Snap
[![snap store badge](https://raw.githubusercontent.com/snapcore/snap-store-badges/master/EN/%5BEN%5D-snap-store-black-uneditable.png)](https://snapcraft.io/edgex-app-rfid-llrp-inventory)

This folder contains snap packaging for the EdgeX Foundry's RFID-LLRP Inventory application service.

The project maintains a rolling release of the snap on the `edge` channel that is rebuilt and published at least once daily.

The snap currently supports both `amd64` and `arm64` platforms.

## Installation

### Installing snapd
The snap can be installed on any system that supports snaps. You can see how to install 
snaps on your system [here](https://snapcraft.io/docs/installing-snapd/6735).

However for full security confinement, the snap should be installed on an 
Ubuntu 18.04 LTS or later (Desktop or Server), or a system running Ubuntu Core 18 or later.

### Installing EdgeX App Service Configurable as a snap
The snap is published in the snap store at https://snapcraft.io/edgex-app-service-configurable.
You can see the current revisions available for your machine's architecture by running the command:

```bash
$ snap info edgex-app-rfid-llrp-inventory
```

The latest stable version of the snap can be installed using:

```bash
$ sudo snap install edgex-app-rfid-llrp-inventory
```

The latest development version of the snap can be installed using:

```bash
$ sudo snap install edgex-app-rfid-llrp-inventory --edge
```

**Note** - the snap has only been tested on Ubuntu Core, Desktop, and Server.

## Using the EdgeX App Service Configurable snap

  
### Configuration Overrides
Configuration changes can be accomplished via the snap's configure hook. If the service has already been started,
updating a setting requires the service to be restarted. 

The following syntax is used to specify service-specific configuration overrides:

```env.<stanza>.<config option>```

For instance, to setup an override of the service's Port use:

```$ sudo snap set env.service.port=2112```

And restart the service:

```$ sudo snap restart edgex-app-rfid-llrp-inventory``

**Note** - at this time changes to configuration values in the [Writable] section are not supported.

For details on the mapping of configuration options to Config options, please refer to "Service Environment Configuration Overrides".

### Startup environment variables

EdgeX services by default wait 60s for dependencies (e.g. Core Data) to become available, and will exit after this time if the dependencies aren't met. The following options can be used to override this startup behavior on systems where it takes longer than expected for the dependent services provided by the edgexfoundry snap to start. Note, both options below are specified as a number of seconds.
    
To change the default startup duration (60 seconds), for a service to complete the startup, aka bootstrap, phase of execution by using the following command:

```bash
$ sudo snap set edgex-app-rfid-llrp-inventory startup-duration=60
```

The following environment variable overrides the retry startup interval or sleep time before a failure is retried during the start-up, aka bootstrap, phase of execution by using the following command:

```bash
$ sudo snap set edgex-app-rfid-llrp-inventory startup-interval=1
```

**Note** - Should the environment variables be modified after the service has started, the service must be restarted.


## Service Environment Configuration Overrides
**Note** - all of the configuration options below must be specified with the prefix: 'env.'

```
[Service]
service.boot-timeout            // Service.BootTimeout
service.health-check-interval   // Service.HealthCheckInterval
service.host                    // Service.Host
service.server-bind-addr        // Service.ServerBindAddr
service.port                    // Service.Port
service.protocol                // Service.Protocol
service.max-result-count        // Service.MaxResultCount
service.max-request-size        // Service.MaxRequestSize
service.startup-msg             // Service.StartupMsg
service.request-timeout         // Service.RequestTimeout

[Clients.core-command]
clients.core-command.port       // Clients.core-command.Port

[Clients.core-data]
clients.core-data.port          // Clients.core-data.Port

[Clients.core-metadata]
clients.core-metadata.port      // Clients.core-metadata.Port

[Clients.support-notifications]
clients.support-notifications.port  // Clients.support-notifications.Port

[Triger]
[Trigger.EdgexMessageBus]
trigger.edgex-message-bus.type                             // Trigger.EdgexMessageBus.Type

[Trigger.EdgexMessageBus.SubscribeHost]
trigger.edgex-message-bus.subscribe-host.port              // Trigger.EdgexMessageBus.SubscribeHost.Port
trigger.edgex-message-bus.subscribe-host.protocol          // Trigger.EdgexMessageBus.SubscribeHost.Protocol
trigger.edgex-message-bus.subscribe-host.subscribe-topics  // Trigger.EdgexMessageBus.SubscribeHost.SubscribeTopics

[Trigger.EdgexMessageBus.PublishHost]
trigger.edgex-message-bus.publish-host.port                // Trigger.EdgexMessageBus.PublishHost.Port
trigger.edgex-message-bus.publish-host.protocol            // Trigger.EdgexMessageBus.PublishHost.Protocol
trigger.edgex-message-bus.publish-host.publish-topic       // Trigger.EdgexMessageBus.PublishHost.PublishTopic


[AppCustom]
  # Every device(reader) + antenna port represents a tag location and can be assigned an alias
  # such as Freezer, Backroom etc. to give more meaning to the data. The default alias set by
  # the application has a format of <deviceName>_<antennaId> e.g. Reader-10-EF-25_1 where
  # Reader-10-EF-25 is the deviceName and 1 is the antennaId.
  # See also: https://github.com/edgexfoundry/app-rfid-llrp-inventory#setting-the-aliases
  #
  # In order to override an alias, set the default alias as the key, and the new alias as the value you want, such as:
  # Reader-10-EF-25_1 = "Freezer"
  # Reader-10-EF-25_2 = "Backroom"
  [AppCustom.Aliases]

  # See: https://github.com/edgexfoundry/app-rfid-llrp-inventory#configuration
  [AppCustom.AppSettings]
  DeviceServiceName = "device-rfid-llrp"
  AdjustLastReadOnByOrigin = true
  DepartedThresholdSeconds = 600
  DepartedCheckIntervalSeconds = 30
  AgeOutHours = 336
  MobilityProfileThreshold = 6.0
  MobilityProfileHoldoffMillis = 500.0
  MobilityProfileSlope = -0.008



```
