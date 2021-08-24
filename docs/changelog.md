# TBD

# 0.3.2
### Features
* Enabled Typescript strict mode for safer code

### Fixes
* Fixed bugs that popped out when strict mode was enabled

# 0.3.1
### Fixes
* Export `TypedServerOverride`

# 0.3.0
### Fixes
* Added a workaround for gRPC's stupid "unimplemented server" requirement, which messes everything up in Typescript

### Breaking Changes
* The Typescript `MinimalGRPCServer` now takes in service registration functions that accept `TypedServerOverride` rather than `grpc.Server`
    * Users should make their server implementation class implement `KnownKeysOnly<ITheUserServiceServer>` rather than `ITheUserServiceServer`, and in their registration functions call `typedServerOverride.addTypedService` instead of `.addService`

# 0.2.2
### Changes
* Switch to using productized docs-checker orb

### Features
* Added a Typescript version

# 0.2.1
### Features
* Added CircleCI checks

# 0.2.0
### Changes
* The Go module is now one layer deeper

### Breaking Changes
* The Go module is now one layer deeper, at `github.com/kurtosis-tech/minimal-grpc-server/golang`
    * Users should append the `/golang` to the end of all their module imports

# 0.1.0
* Initial release
