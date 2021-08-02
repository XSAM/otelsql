# otelsql

[![ci](https://github.com/XSAM/otelsql/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/XSAM/otelsql/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/XSAM/otelsql/branch/main/graph/badge.svg?token=21S08PK9K0)](https://codecov.io/gh/XSAM/otelsql)
[![Go Report Card](https://goreportcard.com/badge/github.com/XSAM/otelsql)](https://goreportcard.com/report/github.com/XSAM/otelsql)
[![Documentation](https://godoc.org/github.com/XSAM/otelsql?status.svg)](https://pkg.go.dev/mod/github.com/XSAM/otelsql)

It is an OpenTelemetry instrumentation for Golang `database/sql`, a port from https://github.com/open-telemetry/opentelemetry-go-contrib/pull/505.

It can only instrument traces for the present.

## Install

```bash
$ go get github.com/XSAM/otelsql
```

## Features

| Feature                    | Description                                                                                                | Status                      | Reason                                                                                                                                                                                      |
| -------------------------- | ---------------------------------------------------------------------------------------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Rows, RowsClose            | If set to true, will enable the creation of spans on corresponding calls.                                  | Enabled by default | We need to know the status of `Rows`                                                                                                                                                        |
| Query                      | If set to true, will enable recording of sql queries in spans.                                             | Enabled by default | `db.statement` will need this, which is a required attribute. https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/database.md |
| Ping                       | If set to true, will enable the creation of spans on Ping requests.                                        | Implemented                    | Ping has context argument, but it might no needs to record.                                                                                                                                 |
| RowsNext                   | If set to true, will enable the creation of events on corresponding calls. This can result in many events. | Implemented                    | It provides more visibility.                                                                                                                                                                |
| DisableErrSkip             | If set to true, will suppress driver.ErrSkip errors in spans.                                              | Implemented                    | ErrSkip error might annoying                                                                                                                                                                |
| AllowRoot                  | If set to true, will allow otelsql to create root spans in absence of existing spans or even context.        | Implemented                     | It might helpful while debugging missing operations. |
| RowsAffected, LastInsertID | If set to true, will enable the creation of spans on RowsAffected/LastInsertId calls.                      | Dropped                     | Don't know its use cases. We might add this later based on the users' feedback.                                                                                                             |
| QueryParams                | If set to true, will enable recording of parameters used with parametrized queries.                        | Dropped                     | It will cause high cardinality values and security problems.                                                                                                                                |

## Example

See [example](./example/main.go)

## Why port this?

Based on [this comment](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/505#issuecomment-800452510), OpenTelemetry SIG team like to see broader usage and community consensus on an approach before they commit to the level of support that would be required of a package in contrib. But it is painful for users without a stable version, and they have to use replacement in `go.mod` to use this instrumentation.

Therefore, I host this module independently for convenience and make improvements based on users' feedback.
