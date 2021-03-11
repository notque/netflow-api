# netflow-api

[![Build Status](https://travis-ci.org/notque/netflow-api.svg?branch=master)](https://travis-ci.org/notque/netflow-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/notque/netflow-api)](https://goreportcard.com/report/github.com/notque/netflow-api)
[![GoDoc](https://godoc.org/github.com/notque/netflow-api?status.svg)](https://godoc.org/github.com/notque/netflow-api)

----

**netflow-api** is a netflow service for OpenStack, originally designed for SAP's internal Openstack Cloud.

# The idea: Netflow for OpenStack

Network traffic in an OpenStack cloud is hidden from the users. netflow-api enables easy access 
to netflow events on a tenant basis, relying on the ELK stack for storage. Now users can view their project level
network events through an API, or as a module in [Elektra](https://github.com/notque/elektra) an OpenStack Dashboard.

----

## Features 

* A managed service for Netflow in OpenStack
* OpenStack Identity v3 authentication and authorization
* Project and domain-level access control (scoping)
* Compatible with other cloud based audit APIs 
* Exposed Prometheus metrics

# Documentation

## For users

* [netflow-api Users Guide](./docs/users/index.md)
* [netflow-api API Reference](./docs/users/netflow-api-v1-reference.md)

## For operators

* [netflow-api Operators Guide](./docs/operators/operators-guide.md)
