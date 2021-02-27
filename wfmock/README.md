# White Flag Mock

This tool mocks a legacy IOTA node providing the `getWhiteFlagConfirmation` API.
When started, it generates milestones confirming migration bundles as specified in the `config.json`.

### Usage

See the `pkg/config/config.go` file for a description of the configuration parameters.

#### Docker

To start the mock providing a new config file and publishing its port use the following:
```
docker run -v ${PWD}/config.json:/app/config.json -p 127.0.0.1:14265:14265 wfmock
```
