# EdgeX Foundry RFID-LLRP Inventory App Service Snap
[![snap store badge](https://raw.githubusercontent.com/snapcore/snap-store-badges/master/EN/%5BEN%5D-snap-store-black-uneditable.png)](https://snapcraft.io/edgex-app-rfid-llrp-inventory)

This folder contains snap packaging for the EdgeX Foundry's RFID-LLRP Inventory application service.

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

## Snap configuration

EdgeX services implement a service dependency check on startup which ensures that all of the runtime dependencies of a particular service are met before the service transitions to active state.

Snapd doesn't support orchestration between services in different snaps. It is therefore possible on a reboot for a device service to come up faster than all of the required services running 
in the main edgexfoundry snap. If this happens, it's possible that the device service repeatedly fails startup, and if it exceeds the systemd default limits, then it might be left in a failed state. 
This situation might be more likely on constrained hardware (e.g. RPi).

This snap therefore implements a basic retry loop with a maximum duration and sleep interval. If the dependent services are not available, the service sleeps for the defined interval (default: 1s) 
and then tries again up to a maximum duration (default: 60s). These values can be overridden with the following commands:
    
To change the maximum duration, use the following command:

```bash
$ sudo snap set edgex-app-rfid-llrp-inventory startup-duration=60
```

To change the interval between retries, use the following command:

```bash
$ sudo snap set edgex-app-rfid-llrp-inventory startup-interval=1
```

The service can then be started as follows. The "--enable" option
ensures that as well as starting the service now, it will be automatically started on boot:

```bash
$ sudo snap start --enable edgex-device-rfid-llrp.device-rfid-llrp
```

### Aliases setup

The `AppConfig.Aliases` setting needs to be provided for the service to work.  See [Setting the Aliases](https://github.com/edgexfoundry/app-rfid-llrp-inventory#setting-the-aliases).

This can either be by

1. using a content interface to provide a `configuration.toml` file with the correct aliases, or

2. during development, set the values manually in Consul. 


### Using a content interface to set device configuration

The `app-config` content interface allows another snap to seed this snap with configuration directories under `$SNAP_DATA/config/app-rfid-llrp-inventory`.

Note that the `app-config` content interface does NOT support seeding of the Secret Store Token because that file is expected at a different path.

Please refer to [edgex-config-provider](https://github.com/canonical/edgex-config-provider), for an example and further instructions.


### Rich Configuration
While it's possible on Ubuntu Core to provide additional profiles via gadget 
snap content interface, quite often only minor changes to existing profiles are required. 

These changes can be accomplished via support for EdgeX environment variable 
configuration overrides via the snap's configure hook.
If the service has already been started, setting one of these overrides currently requires the
service to be restarted via the command-line or snapd's REST API. 
If the overrides are provided via the snap configuration defaults capability of a gadget snap, 
the overrides will be picked up when the services are first started.

The following syntax is used to specify service-specific configuration overrides:


```
env.<stanza>.<config option>
```
For instance, to setup an override of the service's Port use:
```
$ sudo snap set edgex-app-rfid-llrp-inventory env.service.port=2112
```
And restart the service:
```
$ sudo snap restart edgex-device-rfid-llrp.device-rfid-llrp
```

## Service Environment Configuration Overrides

### [Service]
|snap set|configuration.yaml setting|
|---|---|
|env.service.boot-timeout|Service.BootTimeout|
|env.service.health-check-interval|Service.HealthCheckInterval|
|env.service.host|Service.Host|
|env.service.server-bind-addr|Service.ServerBindAddr|
|env.service.port|Service.Port|
|env.service.protocol|Service.Protocol|
|env.service.max-result-count|Service.MaxResultCount|
|env.service.max-request-size|Service.MaxRequestSize|
|env.service.startup-msg|Service.StartupMsg|
|env.service.request-timeout|Service.RequestTimeout|

### [Clients]
|snap set|configuration.yaml setting|
|---|---|
|env.clients.core-command.port|Clients.core-command.Port|
|env.clients.core-data.port|Clients.core-data.Port|
|env.clients.core-metadata.port|Clients.core-metadata.Port|
|env.clients.support-notifications.port|Clients.support-notifications.Port|

### [Trigger]
|snap set|configuration.yaml setting|
|---|---|
|env.trigger.edgex-message-bus.type|Trigger.EdgexMessageBus.Type|
|env.trigger.edgex-message-bus.subscribe-host.port|Trigger.EdgexMessageBus.SubscribeHost.Port|
|env.trigger.edgex-message-bus.subscribe-host.protocol|Trigger.EdgexMessageBus.SubscribeHost.Protocol|
|env.trigger.edgex-message-bus.subscribe-host.subscribe-topics|Trigger.EdgexMessageBus.SubscribeHost.SubscribeTopics|
|env.trigger.edgex-message-bus.publish-host.port|Trigger.EdgexMessageBus.PublishHost.Port|
|env.trigger.edgex-message-bus.publish-host.protocol|Trigger.EdgexMessageBus.PublishHost.Protocol|
|env.trigger.edgex-message-bus.publish-host.publish-topic|Trigger.EdgexMessageBus.PublishHost.PublishTopic|

### [AppCustom]
|snap set|configuration.yaml setting|
|---|---|
|env.appcustom.appsettings.device-service-name|AppCustom.AppSettings.DeviceServiceName|
|env.appcustom.appsettings.adjust-last-read-on-by-origin|AppCustom.AppSettings.AdjustLastReadOnByOrigin|
|env.appcustom.appsettings.departed-threshold-seconds|AppCustom.AppSettings.DepartedThresholdSeconds|
|env.appcustom.appsettings.departed-check-interval-seconds|AppCustom.AppSettings.DepartedCheckIntervalSeconds|
|env.appcustom.appsettings.age-out-hours|AppCustom.AppSettings.AgeOutHours|
|env.appcustom.appsettings.mobility-profile-threshold|AppCustom.AppSettings.MobilityProfileThreshold|
|env.appcustom.appsettings.mobility-profile-holdof-millis|AppCustom.AppSettings.MobilityProfileHoldoffMillis|
|env.appcustom.appsettings.mobility-profile-slope|AppCustom.AppSettings.MobilityProfileSlope|



```
