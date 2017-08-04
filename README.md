# Bosh System Metrics Server Release

This provides bosh health events/metrics over a secure grpc stream. The job in this release is intended to be deployed on the Bosh Director.

## Architecture

![architecture dig][diagram]

### Plugin

The plugin is compatible with [Bosh HM's json plugin][json plugin]. It reads bosh system health events via stdin and streams them to the `Server` via tcp. It is not managed by monit.

### Server

The server listens on tcp localhost for events from the `Plugin`. The server accepts connections from clients such as the [Bosh System Metrics Forwarder][forwarder] and sends the events over secure grpc. Clients need to specify an `authorization` token in the grpc metadata. This must be a valid token issued by the Bosh Director's UAA and include the `bosh.system_metrics.read` authority.

## High Availability

The server distributes the events on a per subscription basis. That is, if two clients connect with the same `subscription-id`, the event stream will be distributed evenly between them. If two clients connect with _different_ `subscription-id`s, then the server will send a copy of each event to each client.

[forwarder]: https://github.com/pivotal-cf/bosh-system-metrics-forwarder-release
[server]: https://github.com/pivotal-cf/bosh-system-metrics-server-release
[json plugin]: https://github.com/cloudfoundry/bosh/blob/262.x/src/bosh-monitor/lib/bosh/monitor/plugins/json.rb
[diagram]: https://docs.google.com/a/pivotal.io/drawings/d/1l1iAQaBc6SHIpWb3x-lI9p4JVIZN_3ErepbAohqnaPw/pub?w=1192&h=719
