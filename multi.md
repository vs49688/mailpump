# MailPump Multi

This document describes MailPump's `multi` mode. In this mode, multiple sources are "pumped" to a single
destination, potentially relieving load on the target server.

## Usage

```
NAME:
   mailpump run-multi - Run the experimental many-to-one pump

USAGE:
   mailpump run-multi [command options] [arguments...]

OPTIONS:
   --config value, -c value  path to configuration file, or '-' to read from stdin (default: "config.json")
```

## Configuration Reference

| Option (JSON Pointer) | Type                                    | Description                       |
|-----------------------|-----------------------------------------|-----------------------------------|
| `/destination`        | [Connection Config](#connection-config) | Destination server configuration. |
| `/source/${name}`     | [Source Config](#source-config)         | Source server configuration.      |

### Source Config

| Option (JSON Pointer)     | Type                                    | Example       | Description                                                                                  |
|---------------------------|-----------------------------------------|---------------|----------------------------------------------------------------------------------------------|
| `/connection`             | [Connection Config](#connection-config) |               | Source server configuration.                                                                 |
| `/target_mailbox`         | string                                  | `INBOX`       | Name of the mailbox on the destination server.                                               |
| `/idle_fallback_interval` | integer, nanoseconds                    | `60000000000` | Fallback poll interval in the event that the server doesn't support IDLE.                    |
| `/batch_size`             | integer                                 | `15`          | No. messages to cache before ingesting.                                                      |
| `/disable_deletions`      | bool                                    | `false`       | Debug flag, disables deletions from the source. Be VERY careful.                             |
| `/fetch_buffer_size`      | integer                                 | `20`          | No. messages to fetch at a time.                                                             |
| `/fetch_max_interval`     | integer, nanoseconds                    | `30000000000` | Interval at which to poll for messages and flush cached messages, regardless of IDLE status. |

### Connection Config

| Option (JSON Pointer) | Type   | Example                     | Description                              |
|-----------------------|--------|-----------------------------|------------------------------------------|
| `/url`                | string | `imaps://imap.gmail.com`    | IMAP Server URL                          |
| `/username`           | string | `joe.bloggs`                | Username                                 |
| `/auth_method`        | string | `LOGIN`                     | See [here](README.md#authentication).    |
| `/password`           | string | `PassW0Rd1`                 | See [here](README.md#authentication).    |
| `/password_file`      | string | `/path/to/my-password`      | See [here](README.md#authentication).    |
| `/systemd_credential` | string | `my-credential-name`        | See below.                               |
| `/tls_skip_verify`    | bool   | `false`                     | Skip TLS peer & hostname verification.   |
| `/transport`          | string | `persistent`, or `standard` | IMAP transport implementation to use.    |
| `/debug`              | bool   | `false`                     | Enable IMAP session debug logging.       |
| `/oauth2`             |        |                             | Temporarily unsupported in `multi`-mode. |

**systemd Note**

MailPump provides support for systemd's `LoadCredential=`. If the `systemd_credential` configuration option
is set, then the provided value will be searched for relative to `$CREDENTIALS_DIRECTORY`.

## Example Configuration

```json
{
  "destination": {
    "systemd_credential": "destination",
    "tls_skip_verify": false,
    "transport": "persistent",
    "url": "imaps://imap.migadu.com",
    "username": "my-user@example.com"
  },
  "sources": {
    "au-com-yahoo-joebloggs": {
      "batch_size": 15,
      "connection": {
        "password": "my-insecure-password",
        "transport": "persistent",
        "url": "imaps://imap.mail.yahoo.com/INBOX",
        "username": "my-user@yahoo.com"
      },
      "fetch_buffer_size": 20,
      "fetch_max_interval": 300000000000,
      "idle_fallback_interval": 60000000000,
      "target_mailbox": "INBOX"
    },
    "au-com-yahoo-vs49688-junk": {
      "batch_size": 15,
      "connection": {
        "password_file": "/path/to/my-password",
        "tls_skip_verify": true,
        "transport": "persistent",
        "url": "imaps://imap.mail.yahoo.com/Bulk",
        "username": "my-user@yahoo.com"
      },
      "fetch_buffer_size": 20,
      "fetch_max_interval": 300000000000,
      "idle_fallback_interval": 60000000000,
      "target_mailbox": "Junk"
    }
  }
}
```