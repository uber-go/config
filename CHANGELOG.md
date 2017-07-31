# Changelog

## v1.0.0-rc2 (unreleased)

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

## v1.0.0-rc1 (26 Jun 2017)

- **[Breaking]** `Provider` interface was trimmed down to 2 methods: `Name` and `Get`