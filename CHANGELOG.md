# Changelog

## [0.7.1](https://github.com/kurtosis-tech/minimal-grpc-server/compare/0.7.0...0.7.1) (2023-03-17)


### Features

* Add possibility to enable HTTPS on GRPC server ([#35](https://github.com/kurtosis-tech/minimal-grpc-server/issues/35)) ([4b160f0](https://github.com/kurtosis-tech/minimal-grpc-server/commit/4b160f0d17db8a7d952e9a5fc6c26251ea176cc2))

## [0.7.0](https://github.com/kurtosis-tech/minimal-grpc-server/compare/0.6.2...0.7.0) (2023-03-16)


### ⚠ BREAKING CHANGES

* RunUntilStopped now accepts a chan <-struct{} instead of chan <-interface{} ([#33](https://github.com/kurtosis-tech/minimal-grpc-server/issues/33))

### Code Refactoring

* RunUntilStopped now accepts a chan &lt;-struct{} instead of chan <-interface{} ([#33](https://github.com/kurtosis-tech/minimal-grpc-server/issues/33)) ([1f3f722](https://github.com/kurtosis-tech/minimal-grpc-server/commit/1f3f72269eacff45582f24cef7524c582011d96a))

## [0.6.2](https://github.com/kurtosis-tech/minimal-grpc-server/compare/0.6.1...0.6.2) (2023-03-16)


### Bug Fixes

* bump ansi-regex from 3.0.0 to 3.0.1 in /typescript ([#23](https://github.com/kurtosis-tech/minimal-grpc-server/issues/23)) ([4adf4f0](https://github.com/kurtosis-tech/minimal-grpc-server/commit/4adf4f054260ba0c8aa1cefb1b86acab4e13d1ab))
* bump json5 from 2.2.0 to 2.2.3 in /typescript ([#28](https://github.com/kurtosis-tech/minimal-grpc-server/issues/28)) ([7b3b453](https://github.com/kurtosis-tech/minimal-grpc-server/commit/7b3b4534e987ddc55879a15e380d1b7f1aa8ee38))
* bump minimist from 1.2.5 to 1.2.7 in /typescript ([#25](https://github.com/kurtosis-tech/minimal-grpc-server/issues/25)) ([c16a423](https://github.com/kurtosis-tech/minimal-grpc-server/commit/c16a423b5d1ca551acee20fb03a3a5d2730cec25))

## 0.6.1

* Migrate repo to use internal tool `kudet` for release process
* Migrate `develop` into `master`
* 
## 0.6.0
### Changes
* Switched the Typescript library to use `@grpc/grpc-js` rather than the `grpc` package, as the `grpc` package is deprecated

### Breaking Changes
* The service interfaces that are passed in to the server must be of type `@grpc/grpc-js` rather than `grpc`

## 0.5.0
### Removals
* Removed the "protocol" arg to the Go library, because gRPC can only run on TCP

### Breaking Changes
* `NewMinimalGRPCServer` constructors in the Go library no longer take in a protocol

## 0.4.0
### Features
* Added a `runUntilStopped` method to the server, allowing the server to be stopped using an arbitrary event (rather than only interrupts)
* Added `runUntilStopped` tests to Golang & Typescript

### Changes
* Renamed `run` to `runUntilInterrupted`

### Fixes
* Fixed a bug where even if the Typescript server shut down correctly, it would still wait for the hard stop timeout
* Fixed a bug in the TS lib where the server would go through the hard stop flow if the server actually stopped correctly, and vice versa

### Breaking Changes
* Renamed `run` to `runUntilInterrupted`
* The Golang library's listen port is now a `uint16`
* The Typescript library now requires Node >= `16.13.0`

## 0.3.8
### Fixes
* `stacktrace.Propagate` panics when receiving a `nil` cause

## 0.3.7
### Features
* Added logging to all requests made to users of the Go version of this library
    * **NOTE:** This is NOT enabled for Typescript because server-side interceptors aren't supported unfortunately: https://github.com/grpc/grpc-node/issues/419

### Fixes
* Added `go build ./...` to Go buildscript, as some compile errors weren't getting caught

## 0.3.6
### Fixes
* Fixed a bug where the gRPC server was binding on the incorrect IP

## 0.3.5
### Fixes
* Fixed a bug where the bind required `127.0.0.1` specified

## 0.3.4
### Fixes
* Fixed a bug in the way we created the insecure server credentials

## 0.3.3
### Fixes
* Unpin Node engine, using `>=14.17.0` now

## 0.3.2
### Features
* Enabled Typescript strict mode for safer code

### Fixes
* Fixed bugs that popped out when strict mode was enabled

## 0.3.1
### Fixes
* Export `TypedServerOverride`

## 0.3.0
### Fixes
* Added a workaround for gRPC's stupid "unimplemented server" requirement, which messes everything up in Typescript

### Breaking Changes
* The Typescript `MinimalGRPCServer` now takes in service registration functions that accept `TypedServerOverride` rather than `grpc.Server`
    * Users should make their server implementation class implement `KnownKeysOnly<ITheUserServiceServer>` rather than `ITheUserServiceServer`, and in their registration functions call `typedServerOverride.addTypedService` instead of `.addService`

## 0.2.2
### Changes
* Switch to using productized docs-checker orb

### Features
* Added a Typescript version

## 0.2.1
### Features
* Added CircleCI checks

## 0.2.0
### Changes
* The Go module is now one layer deeper

### Breaking Changes
* The Go module is now one layer deeper, at `github.com/kurtosis-tech/minimal-grpc-server/golang`
    * Users should append the `/golang` to the end of all their module imports

## 0.1.0
* Initial release
