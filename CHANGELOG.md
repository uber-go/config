# Changelog

## v1.0.3 (2017-08-25)

- Fixed compilation errors for `GOARCH=arm/x86`.

## v1.0.2 (2017-08-17)

- Fixed populate panic for a nil pointer.

## v1.0.1 (2017-08-04)

- Fixed unmarshal text on missing value.

## v1.0.0 (2017-07-31)

First stable release: no breaking changes will be made in the 1.x series.

- **[Breaking]** `ValueType` and `GetType` functionality is removed in favor of using
  `reflect.Kind`.
- Skip populating function and value types instead of reporting errors.
- **[Breaking]** `Value.Timestamp` is private, use Value.LastUpdated instead.
- **[Breaking]** Use semantic version paths for yaml and validator packages.
- Let user to skip loading command line provider via `commandLine` parameter.
- **[Breaking]** Most of the `Provider` constructors return an error instead of panics.
- **[Breaking]** `Value.WithDefault` returns an error when a default can't be used.
- **[Breaking]** Try and As conversion helpers are removed in favor of using
  other cast libraries.
- **[Breaking]** Removed `Value.IsDefault` method.
- **[Breaking]** Removed Load* functions.
- **[Breaking]** Unexport NewYAMLProviderFromReader* functions.
- **[Breaking]** `NewProviderGroup` returns an error.

## v1.0.0-rc1 (2017-06-26)

- **[Breaking]** `Provider` interface was trimmed down to 2 methods: `Name` and `Get`