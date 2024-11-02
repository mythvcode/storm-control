# Configuration parameters

The program can be configured in two ways: by using a YAML configuration file or by using environment variables. If a configuration file is specified like `./storm-control -config config.yaml`, then **environment variables are not used.**

## Example of config file
[Config example](./config_example.yaml)


## Configuration Explanation

Env variable                    | Yaml config equivalent         | default value               | description                                                                            |
---                             |  ---                           |  ---                        | ---                                                                                    |
LOG_LEVEL                       | logger:level                   | debug                       | Storm control log level                                                                |
LOG_FILE                        | logger:file                    |                             | Log file (if not specified when stdout)                                                |
BLOCK_ENABLED                   | watcher:block_delay            | 10                          | Time duration in seconds before the unblock process initiates, after the block action. |
OS_AUTH_URL                     | watcher:block_enabled          | false                       | Enable block action in case of detected storm control                                  |
BLOCK_THRESHOLD                 | watcher:block_threshold        | 100                         | Threshold of broadcast and multicast packets to trigger block action                   |
STATIC_DEV_LIST                 | watcher:device_list            |                             | Static interface list if specified when device_regex is not checked                    |
DEV_REGEX                       | watcher:device_regex           | ^tap.{8}-.{2}$              | Regexp for search interfaces to monitor                                                |
EXPORTER_HOST                   | exporter:host                  | localhost                   | Exporter host to bind                                                                  |
EXPORTER_PORT                   | exporter:port                  | 8080                        | Exporter port to bind                                                                  |
EXPORTER_REQUEST_TIMEOUT        | exporter:request_timeout       | 10                          | Request timeout seconds                                                                |
EXPORTER_TELEMETRY_PATH         | exporter:telemetry_path        | /metrics                    | Exporter telemetry path                                                                |
EXPORTER_ENABLE                 | exporter:enable                | true                        | Enable exporter                                                                        |
EXPORTER_ENABLE_REQUEST_LOGGING | exporter:enable_request_logging| true                        | Activate logging for exporter API requests                                             |
EXPORTER_ENABLE_RUNTIME_METRICS | exporter:enable_runtime_metrics| false                       | Enable collection golang runtime metrics                                               |