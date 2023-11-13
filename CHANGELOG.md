# App RFID LLRP Inventory

## Change Logs for EdgeX Dependencies

- [app-functions-sdk-go](https://github.com/edgexfoundry/app-functions-sdk-go/blob/main/CHANGELOG.md)

## [v3.1.0] Napa - 2023-11-15 (Only compatible with the 3.x releases)


### ✨  Features

- Remove snap packaging ([#238](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/238)) ([ae0fc91…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/ae0fc916ce7d511bfada48b200efa99a99f94c46))
```text

BREAKING CHANGE: Remove snap packaging ([#238](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/238))

```


### ◀️ Revert

- Restore ignored items that were not snap-related ([#241](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/241)) ([df6d7f6…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/df6d7f608fd9c1c4fe6c937888fc634314559cbf))


### ♻ Code Refactoring

- Remove obsolete comments from config file ([#240](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/240)) ([9e1f568…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/9e1f568c9b1455eaf2490256b50d2aac3d1e2766))
- Replace github.com/pkg/errors with errors ([#218](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/218)) ([49917a2…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/49917a2fa1511478922ba9567a0d5bdfd5b8a77e))


### 📖 Documentation

- Replace docs in README with link to docs on docs.edgexfoundry.org ([#248](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/248)) ([03dfe80…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/03dfe80146e416d2b04b2eaefcdaa11753ed70f8))
- Add swagger file ([#244](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/244)) ([21ed4d0…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/21ed4d02aeb24d2ae6dc2d5f32ea5d4fa656c524))


### 👷 Build

- Upgrade to go-1.21, Linter1.54.2 and Alpine 3.18 ([#224](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/224)) ([373aed2…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/373aed2dc61e6e0ec63a26b7343d1577d4fdcb4f))

### 🤖 Continuous Integration

- Add automated release workflow on tag creation ([c895c02…](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/c895c02946e1a59c8ddbef8aecdd97cb56dca6f0))

## [v3.0.1] Minnesota - 2023-07-25 (Only compatible with the 3.x releases)
### Features ✨
Security - Add missing authentication hooks to standard routes (#1447)

BREAKING CHANGE: EdgeX standard routes, except /ping, will require authentication when running in secure mode

## [v3.0.0] Minnesota - 2023-05-31 (Only compatible with the 3.x releases)

### Features ✨

- Remove ZeroMQ MessageBus capability ([#cbfcac4](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/cbfcac44887bc7898a8a00a335521df61c6eaadd))
  ```text
  BREAKING CHANGE: ZeroMQ MessageBus capability no longer available
  ```
- Consume additional level in event publish topic ([#63c8b30](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/63c8b30869f87de451dd33f2aa13687440ae64a8))
  ```text
  BREAKING CHANGE: Inventory events are published using new topic which includes additional level for the service name.
  ```
- Updates for common config ([#0e6798d](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/0e6798df487ae229a4ea3230d61bef5bbf47a589))
  ```text
  BREAKING CHANGE: configuration file changed to remove common config settings
  ```

### Bug Fixes 🐛

- Change subscription topics to receive any event from device LLRP ([#202](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/202)) ([#ad72238](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/ad72238))
- **snap:** Refactor to avoid conflicts with readonly config provider directory ([#163](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/163)) ([#636b604](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/636b604))

### Code Refactoring ♻

- Use latest SDK for flattened config stem ([#004f5d2](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/004f5d27058d063ff3ccf1e62d985591669bdfad))
  ```text
  BREAKING CHANGE: location of service configuration in Consul changed
  ```
- Rename command line flags for the sake of consistency ([#dc56276](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/dc56276b993b2878b0d870e671804c10f96a6178))
  ```text
  BREAKING CHANGE: renamed -c/--confdir to -cd/--configDirand -f/--file to -cf/--configFile
  ```
- Adjust configuration for reworked MessageBus config ([#bbc8cea](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/bbc8cea5256726f873dd73e5987e9fb16baf68a3))
  ```text
  BREAKING CHANGE: MessageBus configuration is now standalone from Trigger
  ```
- Replace internal topics from config with new constants and use base topic  ([#0d101ae](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/0d101ae6821c8ff83b48d3de973276031649eb12))
  ```text
  BREAKING CHANGE: Internal topics no longer configurable, except the base topic. Trigger topics for edgex-messagebus and external-mqtt now directly under Trigger section. All configured topics (subscribe and function pipeline) now automatically have the base topic (default of "edgex/") prepended.
  ```
- Change configuration file format to YAML  ([#926f659](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/926f659ef75c956bce735ba4a63a1e2481fbf915))
  ```text
  BREAKING CHANGE:  Configuration file now uses YAML format, default file name is now configuration.yaml
  ```
- Enable core command via message bus ([#139](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/139)) ([#494ae06](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/494ae06))
- Consume MakeItRun rename to Run ([#188](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/188)) ([#cc44783](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/cc44783))
- Go 1.20 gofmt ([#157](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/157)) ([#968f145](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/968f145))
- **snap:** Drop the support for legacy snap env options ([#350dcbb](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commit/350dcbb98a3b589a77f3df68bd3874cc550526fa))
  ```text
  BREAKING CHANGE: Drop the support for deprecated snap options starting with `env.`
  ```
- **snap:** Update command and metadata sourcing ([#162](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/162)) ([#0370fe2](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/0370fe2))

### Documentation 📖

- Add main branch Warning ([#191](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/191)) ([#583b590](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/583b590))

### Build 👷

- Ignore all go-mod deps, except go-mod-bootstrap ([#185](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/185)) ([#f3383ef](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/f3383ef))
- Update to Go 1.20, Alpine 3.17 and linter v1.51.2 ([#158](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/158)) ([#9fc1e83](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/9fc1e83))

## [v2.3.0] - Levski - 2022-11-09 (Only compatible with the 2.x releases)

### Features ✨

- Add common service metrics configuration ([#118](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/118)) ([#76318d8](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/76318d8))
- Add NATS configuration ([#109](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/109)) ([#d157eb8](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/d157eb8))
- **snap:** add config interface with unique identifier ([#115](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/115)) ([#617f1cb](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/617f1cb))

### Documentation

- Update attribution.txt to reference paho license as v2.0 ([#89](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/89)) ([#cd50c67](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/cd50c67))

### Code Refactoring ♻

- **snap:** edgex-snap-hooks related upgrade ([#101](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/101)) ([#dc12914](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/dc12914))

### Build 👷

- Add option to build Service with NATS Capability ([#109](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/109)) ([#d157eb8](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/d157eb8))
- Upgrade to Go 1.18 and alpine 3.16 ([#96](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/96)) ([#ccdb054](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/ccdb054))
- Update attribution script to use go.mod file instead of vendor folder ([#95](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/95)) ([#5d59561](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/5d59561))

## [v2.2.0] - Kamakura - 2022-05-11 (Only compatible with the 2.x releases)

### Features ✨

- enable security hardening ([#67](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/67)) ([#5dcf95f](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/5dcf95f))

- Update to latest go-mod-messaging w/o ZMQ on windows ([#1009](https://github.com/edgexfoundry/app-functions-sdk-go/issues/1009)) ([#d30acd6](https://github.com/edgexfoundry/app-functions-sdk-go/commits/d30acd6))

  ```
  BREAKING CHANGE:
  ZeroMQ no longer supported on native Windows for EdgeX
  MessageBus
  ```

### Documentation 📖

- **snap:** Move usage instructions to docs ([#79](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/79)) ([#9387e44](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/9387e44))

## [v2.1.0] - Jakarta - 2022-04-27 (Only compatible with the 2.x releases)

### Features ✨
- Migrate service to V2 ([#54](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/54)) ([#73352f1](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/73352f1))
### Build 👷
- update alpine base to 3.14 ([#51](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/51)) ([#e04a038](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/e04a038))

- Added "make lint" target  and added it to "make test". Also resolved all lint errors ([#63](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/63)) ([#a218d4f](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/a218d4f))

  <a name="v1.0.0"></a>

## [v1.0.0] - 2021-08-20
### Bug Fixes 🐛
- Retry failed HTTP GET calls ([#47](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/47)) ([#088d1eb](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/088d1eb))
- Correct parsing of ImpinjPeakRSSI parameter ([#44](https://github.com/edgexfoundry/app-rfid-llrp-inventory/issues/44)) ([#0853f1f](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/0853f1f))
### Code Refactoring ♻
- Clean up TOML quotes and add LF MD files ([#89e3554](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/89e3554))
### Documentation 📖
- Add badges to readme ([#9082428](https://github.com/edgexfoundry/app-rfid-llrp-inventory/commits/9082428))
