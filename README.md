# ancla

ancla does something good.

[![Build Status](https://travis-ci.com/xmidt-org/ancla.svg?branch=main)](https://travis-ci.com/xmidt-org/ancla)
[![codecov.io](http://codecov.io/github/xmidt-org/ancla/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/ancla?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/ancla)](https://goreportcard.com/report/github.com/xmidt-org/ancla)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/ancla/blob/main/LICENSE)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_PROJECT&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_ancla)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/ancla.svg)](CHANGELOG.md)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xmidt-org/ancla)](https://pkg.go.dev/github.com/xmidt-org/ancla)

## Setup

1. Search and replace ancla with your project name.
1. Initialize `go.mod` file: `go mod init github.com/xmidt-org/ancla`
1. Add org teams to project (Settings > Manage Access): 
    - xmidt-org/admins with Admin role
    - xmidt-org/server-writers with Write role
1. Manually create the first release.  After v0.0.1 exists, other releases will be made by automation after the CHANGELOG is updated to reflect a new version header and nothing under the Unreleased header.
1. For libraries:
    1. Add org workflows in dir `.github/workflows`: push, tag, and release. This can be done by going to the Actions tab for the repo on the github site.
    1. Remove the following files/dirs: `.dockerignore`, `Dockerfile`, `Makefile`, `rpkg.macros`, `ancla.yaml`, `deploy/`, and `conf/`.
1. For applications:
    1. Remove PkgGoDev badge from this file.
    1. Add org workflows in dir `.github/workflows`: push, tag, release, and docker-release. This can be done by going to the Actions tab for the repo on the github site.
    1. Add project name, `.ignore`, and `errors.txt` to `.gitignore` file.
    1. Update `Dockerfile` - choose new ports to expose that no current XMiDT application is using.
    1. Update `deploy/packaging/ancla.spec` file to have a proper Summary and Description.
    1. Update `conf/ancla.service` file to have a proper Description.


## Summary

Summary should be a small paragraph explanation of what this project does.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Details](#details)
- [Install](#install)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/). 
By participating, you agree to this Code.

## Details

Add details here.

## Install

Add details here.

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
