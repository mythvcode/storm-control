# Metrics Description

Label `traffic_type` can have the following values:
- `ipv4_multicast`
- `ipv6_multicast`
- `other_multicast`
- For metric `storm_control_traffic_blocked_status` value can be also `broadcast`


| Metric                                            | Labels                                              | Type    | Description                                                                                   |
| ---                                               | ---                                                 | ---     | ---                                                                                           |
| `storm_control_list_attached_interfaces`          | `interface_index`, `interface_name`                 | gauge   | Metric shows the list of attached interfaces, value is always 1                               |
| `storm_control_traffic_blocked_status`            | `interface_index`, `interface_name`, `traffic_type` | counter | Block status of a specific type of traffic on a specific interface (1 blocked, 2 not blocked) |
| `storm_control_broadcast_dropped_packets`         | `interface_index`, `interface_name`                 | counter | Number of dropped broadcast packets for a specific interface                                  |
| `storm_control_broadcast_passed_packets`          | `interface_index`, `interface_name`                 | counter | Number of passed broadcast packets for a specific                                             |
| `storm_control_multicast_passed_packets_by_type`  | `interface_index`, `interface_name`, `traffic_type` | counter | Number of passed multicast packets for a specific interface (grouped by traffic type)         |
| `storm_control_multicast_dropped_packets_by_type` | `interface_index`, `interface_name`, `traffic_type` | counter | Number of dropped multicast packets for a specific interface (grouped by traffic type)        |
| `storm_control_multicast_passed_packets_total`    | `interface_index`, `interface_name`                 | counter | Total number of passed multicast packets for a specific interface                             |
| `storm_control_multicast_dropped_packets_total`   | `interface_index`, `interface_name`                 | counter | Total number of dropped multicast packets for a specific interface                            |