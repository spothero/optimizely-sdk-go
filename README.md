# optimizely-sdk-go
[![GoDoc](https://godoc.org/github.com/spothero/optimizely-sdk-go?status.svg)](https://godoc.org/github.com/spothero/optimizely-sdk-go)
[![Build Status](https://circleci.com/gh/spothero/optimizely-sdk-go/tree/master.svg?style=shield)](https://circleci.com/gh/spothero/optimizely-sdk-go/tree/master)
[![codecov](https://codecov.io/gh/spothero/optimizely-sdk-go/branch/master/graph/badge.svg)](https://codecov.io/gh/spothero/optimizely-sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/spothero/optimizely-sdk-go)](https://goreportcard.com/report/github.com/spothero/optimizely-sdk-go)

This project aims to provide an SDK for interacting with [Optimizely full stack](https://optimizely.com) in Go.

This project is in early development stages and does not implement the full feature-set of the official Optimizely SDKs
for other languages nor is it officially endorsed by Optimizely. PRs for feature additions and bug fixes are welcome!

Supported features:

- [x] [Basic A/B test bucketing](https://docs.developers.optimizely.com/full-stack/docs/run-a-b-tests)
- [x] Impression reporting
- [x] Read Projects, Environments, and Datafiles from the REST API
- [ ] [Audiences](https://docs.developers.optimizely.com/full-stack/docs/define-audiences-and-attributes)
- [ ] [Mutual Exclusion](https://docs.developers.optimizely.com/full-stack/docs/use-mutual-exclusion)
- [ ] [Feature tests](https://docs.developers.optimizely.com/full-stack/docs/run-feature-tests)
