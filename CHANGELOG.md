# App RFID LLRP Inventory

## Change Logs for EdgeX Dependencies

- [app-functions-sdk-go](https://github.com/edgexfoundry/app-functions-sdk-go/blob/main/CHANGELOG.md)

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
