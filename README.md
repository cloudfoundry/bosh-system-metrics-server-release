**Deprecated**: Replaced by [system-metrics](https://github.com/cloudfoundry/system-metrics-release) and [system-metrics-scraper](https://github.com/cloudfoundry/system-metrics-scraper-release).

# Bosh System Metrics Server Release

This provides bosh health events (heartbeats and alerts) over a secure grpc stream. The job in this release is intended to be deployed on the Bosh Director. For more info, see [wiki](https://github.com/cloudfoundry/bosh-system-metrics-server-release/wiki)

If you have any questions, or want to get attention for a PR or issue please reach out on the [#logging-and-metrics channel in the cloudfoundry slack](https://cloudfoundry.slack.com/archives/CUW93AF3M)

## Architecture

![architecture dig][diagram]

### Plugin

The plugin is compatible with [Bosh HM's json plugin][json plugin]. It reads bosh system health events via stdin and streams them to the **Server** via tcp. It is not managed by monit. However, if this plugin does fail, bosh HM's json plugin will restart it.

### Server

The server listens on tcp localhost for events from the **Plugin**. The server accepts connections from clients such as the [Bosh System Metrics Forwarder][forwarder] and sends the events over secure grpc. Clients need to specify an _authorization_ token in the grpc metadata. This must be a valid token issued by the Bosh Director's UAA and include the `bosh.system_metrics.read` authority.

## High Availability

The server distributes the events on a subscription basis. That is, if two clients connect with the same `subscription-id`, the event stream will be distributed evenly between them. If two clients connect with _different_ `subscription-id`s, they will each get a copy of the event stream.

[forwarder]: https://github.com/cloudfoundry/bosh-system-metrics-forwarder-release
[server]: https://github.com/cloudfoundry/bosh-system-metrics-server-release
[json plugin]: https://github.com/cloudfoundry/bosh/blob/262.x/src/bosh-monitor/lib/bosh/monitor/plugins/json.rb
[diagram]: https://docs.google.com/a/pivotal.io/drawings/d/1l1iAQaBc6SHIpWb3x-lI9p4JVIZN_3ErepbAohqnaPw/pub?w=1192&h=719
