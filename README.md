# :fishing_pole_and_fish: config [![GoDoc][doc-img]][doc] [![GitHub release][release-img]][release] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Report Card][report-card-img]][report-card]

Package `config` allows users to:

* Get components working with minimal configuration
* Override any field if the default doesn't make sense for their use case

## Installation
We recommend locking to [SemVer](http://semver.org/) range `^1` using
[Glide](https://github.com/Masterminds/glide):

```
glide get 'go.uber.org/config#^1
```

## Stability

This library is `v1` and follows [SemVer](http://semver.org/) strictly.

No breaking changes will be made to exported APIs before `v2.0.0`.

[doc-img]: http://img.shields.io/badge/GoDoc-Reference-blue.svg
[doc]: https://godoc.org/go.uber.org/config

[release-img]: https://img.shields.io/github/release/uber-go/config.svg
[release]: https://github.com/uber-go/config/releases

[ci-img]: https://img.shields.io/travis/uber-go/config/master.svg
[ci]: https://travis-ci.org/uber-go/config/branches

[cov-img]: https://codecov.io/gh/uber-go/config/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/config/branch/master

[report-card]: https://goreportcard.com/report/github.com/uber-go/config
[report-card-img]: https://goreportcard.com/badge/github.com/uber-go/config
