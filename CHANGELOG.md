# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

- No changes yet.

## v1.1.0 - 2017-09-28
### Added
- Make expand functions transform a special sequence $$ to literal $.
- Export `Provider` constructors that take `io.Reader`.

### Fixed
- Determine the types of objects encapsulated by `config.Value` with the YAML
  unmarshaller regardless of whether expansion was performed or not.

## v1.0.2 - 2017-08-17
### Fixed
- Fix populate panic for a nil pointer.

## v1.0.1 - 2017-08-04
### Fixed
- Fix unmarshal text on missing value.

## v1.0.0 - 2017-07-31
### Changed
- Skip populating function and value types instead of reporting errors.
- Return an error from provider constructors instead of panic'ing.
- Return an error from `Value.WithDefault` if the default is unusable.

### Removed
- Remove timestamps on `Value`.
- Remove `Try` and `As` conversion helpers.
- Remove `Value.IsDefault` method.
- Remove `Load` family of functions.
- Unexport `NewYAMLProviderFromReader` family of functions.

### Fixed
- Use semantic version paths for yaml and validator packages.

## v1.0.0-rc1 - 2017-06-26
### Removed
- Trim `Provider` interface down to just `Name` and `Get`.
