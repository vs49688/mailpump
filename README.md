# MailPump

A service that monitors a mailbox for messages and will automatically move them
to another, usually on a different server.

Used to liberate your mail from proprietary providers that don't provide automatic
forward-and-delete functionality, i.e. most of them.

## Usage
```
NAME:
   mailpump run - Run the pump

USAGE:
   mailpump run [command options] [arguments...]

OPTIONS:
   --source-url value                   source url [$MAILPUMP_SOURCE_URL]
   --source-auth-method value           source auth method (default: "normal") [$MAILPUMP_SOURCE_AUTH_METHOD]
   --source-username value              source imap username [$MAILPUMP_SOURCE_USERNAME]
   --source-password value              source imap password [$MAILPUMP_SOURCE_PASSWORD]
   --source-password-file value         source imap password file [$MAILPUMP_SOURCE_PASSWORD_FILE]
   --source-tls-skip-verify             skip source tls verification (default: false) [$MAILPUMP_SOURCE_TLS_SKIP_VERIFY]
   --source-transport value             source imap transport (persistent, standard) (default: "persistent") [$MAILPUMP_SOURCE_TRANSPORT]
   --source-debug value                 display source debug info (default: "persistent") [$MAILPUMP_SOURCE_DEBUG]
   --source-oauth2-provider value       source oauth2 provider (custom, google) (default: "custom") [$MAILPUMP_SOURCE_OAUTH2_PROVIDER]
   --source-oauth2-client-id value      source oauth2 client id [$MAILPUMP_SOURCE_OAUTH2_CLIENT_ID]
   --source-oauth2-client-secret value  source oauth2 client secret [$MAILPUMP_SOURCE_OAUTH2_CLIENT_SECRET]
   --source-oauth2-token-url value      source oauth2 token url [$MAILPUMP_SOURCE_OAUTH2_TOKEN_URL]
   --source-oauth2-scopes value         source oauth2 scopes [$MAILPUMP_SOURCE_OAUTH2_SCOPES]
   --dest-url value                     dest url [$MAILPUMP_DEST_URL]
   --dest-auth-method value             dest auth method (default: "normal") [$MAILPUMP_DEST_AUTH_METHOD]
   --dest-username value                dest imap username [$MAILPUMP_DEST_USERNAME]
   --dest-password value                dest imap password [$MAILPUMP_DEST_PASSWORD]
   --dest-password-file value           dest imap password file [$MAILPUMP_DEST_PASSWORD_FILE]
   --dest-tls-skip-verify               skip dest tls verification (default: false) [$MAILPUMP_DEST_TLS_SKIP_VERIFY]
   --dest-transport value               dest imap transport (persistent, standard) (default: "persistent") [$MAILPUMP_DEST_TRANSPORT]
   --dest-debug value                   display dest debug info (default: "persistent") [$MAILPUMP_DEST_DEBUG]
   --dest-oauth2-provider value         dest oauth2 provider (custom, google) (default: "custom") [$MAILPUMP_DEST_OAUTH2_PROVIDER]
   --dest-oauth2-client-id value        dest oauth2 client id [$MAILPUMP_DEST_OAUTH2_CLIENT_ID]
   --dest-oauth2-client-secret value    dest oauth2 client secret [$MAILPUMP_DEST_OAUTH2_CLIENT_SECRET]
   --dest-oauth2-token-url value        dest oauth2 token url [$MAILPUMP_DEST_OAUTH2_TOKEN_URL]
   --dest-oauth2-scopes value           dest oauth2 scopes [$MAILPUMP_DEST_OAUTH2_SCOPES]
   --log-level value                    log level (default: "info") [$MAILPUMP_LOG_LEVEL]
   --log-format value                   log format (text/json) (default: "text") [$MAILPUMP_LOG_FORMAT]
   --idle-fallback-interval value       fallback poll interval for servers that don't support IDLE (default: 1m0s) [$MAILPUMP_IDLE_FALLBACK_INTERVAL]
   --batch-size value                   deletion batch size (default: 15) [$MAILPUMP_BATCH_SIZE]
   --fetch-buffer-size value            fetch buffer size (default: 20) [$MAILPUMP_FETCH_BUFFER_SIZE]
   --fetch-max-interval value           maximum interval between fetches. can abort IDLE (default: 5m0s) [$MAILPUMP_FETCH_MAX_INTERVAL]
   --help, -h                           show help (default: false)
```

## Authentication

MailPump supports three authentication methods: `LOGIN`, `PLAIN`, and `OAUTHBEARER`, which should be passed to
the `--source-auth-method` and `--dest-auth-method` parameters:

### LOGIN

The `LOGIN` authentication method corresponds to the IMAP LOGIN command[^rfc3501].

Example parameters:

| Parameter       | Value        | Required | 
|-----------------|--------------|----------|
| `*-auth-method` | `LOGIN`      | Yes      |
| `*-username`    | `joe.bloggs` | Yes      |
| `*-password`    | `PassW0Rd1`  | Yes      |

[^rfc3501]: https://datatracker.ietf.org/doc/html/rfc3501#section-6.2.3

### PLAIN

The `PLAIN` authentication method corresponds to the SASL PLAIN mechanism[^rfc4616].

[^rfc4616]: https://datatracker.ietf.org/doc/html/rfc4616

Example parameters:

| Parameter       | Value        | Required |
|-----------------|--------------|----------|
| `*-auth-method` | `PLAIN`      | Yes      |
| `*-username`    | `joe.bloggs` | Yes      |
| `*-password`    | `PassW0Rd1`  | Yes      |

### OAUTHBEARER

The `OAUTHBEARER` authentication method corresponds to the SASL OAUTHBEARER mechanism[^rfc7628].

[^rfc7628]: https://datatracker.ietf.org/doc/html/rfc7628

MailPump has in-built support for Google, however can also be configured to
use a custom OAuth2 provider. More in-built providers may be added in future releases.

| Parameter                      | Example                                   | Required                      |
|--------------------------------|-------------------------------------------|-------------------------------|
| `*-oauth2-provider`            | `google,custom`                           | Yes                           |
| `*-oauth2-client-id`           | `mailpump`                                | If `*-oauth2-provider=custom` |
| `*-oauth2-client-secret`       | `d2baf3d2-5810-4dd1-afec-3a0101b28980`    | If `*-oauth2-provider=custom` |
| `*-oauth2-token-url`           | `https://server.example.com/oauth2/token` | If `*-oauth2-provider=custom` |
| `*-oauth2-scopes`[^oauthmulti] | `imap`                                    | If `*-oauth2-provider=custom` |
| `*-username`                   | `joe.bloggs`                              | Yes                           |
| `*-password`[^oauthrefresh]    | `b2F1dGgyLXJlZnJlc2gtdG9rZW4K`            | Yes                           |

[^oauthmulti]: This may be specified multiple times to add multiple scopes.
  When configuring via environment variable, separate the values with commas.

[^oauthrefresh]: This should be a base64-encoded OAuth2 Refresh Token.

## OAuth2 Login

The `mailpump oauthlogin` command will initiate an OAuth2 login and print a token suitable for passing to
`*-password`.

```
NAME:
   mailpump oauthlogin - Generate an OAuth2 Token

USAGE:
   mailpump oauthlogin [command options] [arguments...]

OPTIONS:
   --provider value       provider (custom, google) (default: "custom") [$MAILPUMP_PROVIDER]
   --client-id value      client id [$MAILPUMP_CLIENT_ID]
   --client-secret value  client secret [$MAILPUMP_CLIENT_SECRET]
   --token-url value      token url [$MAILPUMP_TOKEN_URL]
   --scopes value         scopes [$MAILPUMP_SCOPES]
   --help, -h             show help (default: false)
```

## Provider URL Examples

| Provider | URL                                      |
|----------|------------------------------------------|
| Generic  | `imap[s]://hostname[:port]/mailbox/path` |
| Migadu   | `imaps://imap.migadu.com/INBOX`          |
| Yahoo!   | `imaps://imap.mail.yahoo.com/INBOX`      |
| Outlook  | `imaps://outlook.office365.com/INBOX`    |
| GMail    | `imaps://imap.gmail.com/INBOX`           |

## License

Copyright &copy; 2022 [Zane van Iperen](mailto:zane@zanevaniperen.com)

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License version 2, and only
version 2 as published by the Free Software Foundation.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
