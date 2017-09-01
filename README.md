# Hermes

Hermes is an OpenStack audit data retrieval service for accessing CADF audit events collected through Keystone middleware.   
 
At SAP we collect CADF audit events from our Converged Cloud platform and provide them as an auditing service for our users.  

It is named after the Futurama character, not the Greek god.

## Features 

Hermes Service (TODO add operators guide)
* OpenStack Identity v3 authentication and authorization
* Project and domain-level access control (scoping)
* Compatible with other cloud based audit APIs 

# Installation 

Via Makefule:

* `make` to just compile and run the binaries from the `build/` directory
* `make && make install` to install to `/usr`
* `make && make install PREFIX=/some/path` to install to `/some/path`
* `make docker` to build the Docker image (set image name and tag with the `DOCKER_IMAGE` and `DOCKER_TAG` variables)


# Using Hermes

Hermes can be used via our [Converged Cloud Dashboard Elektra](https://github.com/sapcc/elektra) 
via "Auditing" or as an API.

Enter `hermes --help` to see a list of commands and options

Please refer to the Hermes user guide (TODO) for more instructions

# Operating Hermes

The easiest way to deploy Hermes is as a service in Kubernetes.

Feel free to use our [Hermes helm chart](https://github.com/sapcc/helm-charts/tree/master/openstack/hermes)
which contains Hermes deployment, and an ELK deployment for adding Project ID and Domain ID to CADF events generated by Keystone Middleware.

## List events

**GET /v1/events**

Lists a project’s or domain's audit events. The project or domain comes from the 
scope of the authentication token, but can be overriden by a request parameter.

The list of events can be filtered by the parameters passed in via the URL.

Only basic event data will be listed here (event id, event type, resource id,
resource type, user name). Clients must make a separate call to retrieve the full 
CADF payload data for each individual event.

The website for CADF is [here](http://www.dmtf.org/standards/cadf).
More details on the CADF format, with examples for each OpenStack service, can
be found in the PDF 
[here](http://www.dmtf.org/sites/default/files/standards/documents/DSP2038_1.1.0.pdf).

**Parameters**

| **Name** | **Type** | **Description** |
| --- | --- | --- |
| source | string | Selects all events with source similar to this value. |
| resource\_type | string | Selects all events with resource type similar to this value. |
| user\_name | string | Selects all events with user name equal to this value. Prefix matching enabled.|
| event\_type | string | Selects all events with event\_type equal to this value. |
| time | string | Date filter to select all events with _event_time_ matching the specified criteria. See Date Filters below for more detail. |
| offset | integer | The starting index within the total list of the events that you would like to retrieve. |
| limit | integer | The maximum number of records to return (up to 100). The default limit is 10. |
| sort | string | Determines the sorted order of the returned list. See Sorting below for more detail. |
| domain\_id | string | Selects all events in this domain. |
| project\_id | string | Selects all events in this project. |

**Scope:**

If `domain_id` is specified, only events for that domain (at domain level, e.g. project creation) will be returned.

If `project_id` is specified, only events for that project will be returned.

If *both* are specified, *no events will be returned*.

If neither is specified, then the scope of the client's X-Auth-Token will be used.

**Date Filters:**

The value for the `time` parameter is a comma-separated list of time stamps in ISO 
8601 format. The time stamps can be prefixed with any of these comparison operators:
`gt:` (greater-than), `gte:` (greater-than-or-equal), `lt:` (less-than), `lte:` 
(less-than-or-equal).

For example, to get a list of events from May of 2017:
```
GET /v1/events?time=gte:2017-05-01T00:00:00,lt:2017-06-01T00:00:00
```

**Sorting:**

The value of the sort parameter is a comma-separated list of sort keys. Supported 
sort keys include `time`, `source`, `resource_type`, and `event_type`.

Each sort key may also include a direction. Supported directions are `:asc` for 
ascending and `:desc` for descending. The service will use `:asc` for every key 
that does not include a direction.

For example, to sort the list from most recently created to oldest:

```
GET /v1/events?sort=time:desc
```

**Request:**

```
GET /v1/events?offset=1&limit=2&sort=time

Headers:
    Accept: application/json
    X-Auth-Token: {keystone_token}
```

**Response:**

This example shows the audit events for creating & deleting a project.

```json
{
  "next": "http://{hermes_host}:8788/v1/events?limit=2&offset=3",
  "previous": "http://{hermes_host}:8788/v1/events?limit=2&offset=0",
  "events": [
    {
      "source": "identity",
      "event_id": "3824e534-6cd4-53b2-93d4-33dc4ab50b8c",
      "event_type": "identity.project.created",
      "event_time": "2017-04-20T11:27:15.834562+0000",
      "resource_id": "3a7e3d2421384f56a8fb6cf082a8efab",
      "resource_type": "data/security/project",
      "initiator": {
        "domain_id": "39a253e16e4a4a3686edca72c8e101bc",
        "typeURI": "service/security/account/user",
        "user_id": "275e9a16294b3805c8dd2ab77123531af6aacd92182ddcd491933e5c09864a1d",
        "host": {
           "agent": "python-keystoneclient",
           "address": "100.66.0.24"
        },
        "id": "493b9a5284675cbb9f3f6439bd222eb6"
      }
    },
    {
      "source": "identity",
      "event_id": "1ff4703a-d8c3-50f8-94d1-8ab382941e80",
      "event_type": "identity.project.deleted",
      "event_time": "2017-04-20T11:28:32.521298+0000",
      "resource_id": "3a7e3d2421384f56a8fb6cf082a8efab",
      "resource_type": "data/security/project",
      "initiator": {
        "domain_id": "39a253e16e4a4a3686edca72c8e101bc",
        "typeURI": "service/security/account/user",
        "user_id": "275e9a16294b3805c8dd2ab77123531af6aacd92182ddcd491933e5c09864a1d",
        "host": {
           "agent": "python-keystoneclient",
           "address": "100.66.0.24"
        },
        "id": "493b9a5284675cbb9f3f6439bd222eb6"

      }
    }
  ],
  "total": 5
}
```

**Response Attributes**

| **Name** | **Type** | **Description** |
| --- | --- | --- |
| events | list | Contains a list of events. The attributes in the event objects are the same as for an individual event. |
| total | integer | The total number of events available to the user. |
| next | string | A HATEOAS URL to retrieve the next set of events based on the offset and limit parameters. This attribute is only available when the total number of events is greater than offset and limit parameter combined. |
| previous | string | A HATEOAS URL to retrieve the previous set of events based on the offset and limit parameters. This attribute is only available when the request offset is greater than 0. |

**HTTP Status Codes**

| **Code** | **Description** |
| --- | --- |
| 200 | Successful Request |
| 401 | Invalid/expired X-Auth-Token or the token doesn&#39;t have permissions to this resource |

## Event details

**GET /v1/events/<event_id>**

Returns the full CADF payload for an individual
event, e.g.:

```json
{
   "publisher_id": "identity.keystone-2031324599-cgpyi",
   "event_type": "identity.project.deleted",
   "payload": {
      "observer": {
         "typeURI": "service/security",
         "id": "3824e534-6cd4-53b2-93d4-33dc4ab50b8c"
      },
      "resource_info": "d2eec974d849446da1715923e60d0b3b",
      "typeURI": "http://schemas.dmtf.org/cloud/audit/1.0/event",
      "initiator": {
         "domain_id": "39a253e16e4a4a3686edca72c8e101bc",
         "typeURI": "service/security/account/user",
         "user_id": "275e9a16294b3805c8dd2ab77123531af6aacd92182ddcd491933e5c09864a1d",
         "host": {
            "agent": "python-keystoneclient",
            "address": "100.66.0.24"
         },
         "id": "493b9a5284675cbb9f3f6439bd222eb6"
      },
      "eventTime": "2017-04-20T11:28:32.521298+0000",
      "action": "deleted.project",
      "eventType": "activity",
      "id": "1ff4703a-d8c3-50f8-94d1-8ab382941e80",
      "outcome": "success",
      "target": {
         "typeURI": "data/security/project",
         "id": "d2eec974d849446da1715923e60d0b3b"
      }
   },
   "message_id": "d4f88c45-5fea-4013-80ec-2d357eab37c3",
   "priority": "info",
   "timestamp": "2017-04-20 11:28:32.521769"
}
```

## Attributes

**GET /v1/attributes/<attribute_name>**

Returns the unique attributes from ElasticSearch for a given attribute name.

```json 
[
  "identity.role_assignment.created",
  "identity.OS-TRUST:trust.created"
]
```

## Audit Config

**GET /v1/audit/**

Returns the details of configuration for a given audit tenant, e.g.:

```json 
{
    "enabled": true/false,
}
```

**PUT /v1/audit/**

Sets the details of configuration for a given audit tenant, e.g.:

```json 
{
    "tenant_id": "openstack_tenant_string",
    "enabled": true/false
}
```

## Usage
 *TO DO

1. Write a configuration file for your environment, by following the [example configuration][ex-conf].

[ex-conf]:  ./etc/hermes.conf


[![Build Status](https://travis-ci.org/sapcc/hermes.svg?branch=master)](https://travis-ci.org/sapcc/hermes)
[![Coverage Status](https://coveralls.io/repos/github/sapcc/hermes/badge.svg?branch=master)](https://coveralls.io/github/sapcc/hermes?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/sapcc/hermes)](https://goreportcard.com/report/github.com/sapcc/hermes)
[![GoDoc](https://godoc.org/github.com/sapcc/hermes?status.svg)](https://godoc.org/github.com/sapcc/hermes)
