# MailPump

A service that monitors a mailbox for messages and will automatically move them
to another, usually on a different server.

Used to liberate your mail from proprietary providers that don't provide automatic
forward-and-delete functionality, i.e. most of them.

## Usage
```
NAME:
   mailpump - ./mailpump

USAGE:
   mailpump [global options] command [command options] [arguments...]

DESCRIPTION:
   MailPump monitors a mailbox via IMAP and will "pump" mail
   to another mailbox on a different server, deleting the originals.

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --source-url value            source imap url [$MAILPUMP_SOURCE_URL]
   --source-username value       destination imap username [$MAILPUMP_SOURCE_USERNAME]
   --source-password value       source imap password [$MAILPUMP_SOURCE_PASSWORD]
   --source-password-file value  source imap password file [$MAILPUMP_SOURCE_PASSWORD_FILE]
   --source-tls-skip-verify      skip source tls verification (default: false) [$MAILPUMP_SOURCE_TLS_SKIP_VERIFY]
   --source-transport value      source imap transport (persistent, standard) (default: "persistent") [$MAILPUMP_SOURCE_TRANSPORT]
   --source-debug                display source debug info (default: false) [$MAILPUMP_SOURCE_DEBUG]
   --dest-url value              destination imap url [$MAILPUMP_DEST_URL]
   --dest-username value         destination imap username [$MAILPUMP_DEST_USERNAME]
   --dest-password value         destination imap password [$MAILPUMP_DEST_PASSWORD]
   --dest-password-file value    destination imap password file [$MAILPUMP_DEST_PASSWORD_FILE]
   --dest-tls-skip-verify        skip destination tls Verification (default: false) [$MAILPUMP_DEST_TLS_SKIP_VERIFY]
   --dest-transport value        destination imap transport (persistent, standard) (default: "persistent") [$MAILPUMP_DEST_TRANSPORT]
   --dest-debug                  display destination debug info (default: false) [$MAILPUMP_DEST_DEBUG]
   --log-level value             logging level (default: "info") [$MAILPUMP_LOG_LEVEL]
   --log-format value            logging format (text/json) (default: "text") [$MAILPUMP_LOG_FORMAT]
   --tick-interval value         tick interval (default: 1m0s) [$MAILPUMP_TICK_INTERVAL]
   --batch-size value            deletion batch size (default: 15) [$MAILPUMP_BATCH_SIZE]
   --fetch-buffer-size value     fetch buffer size (default: 20) [$MAILPUMP_FETCH_BUFFER_SIZE]
   --help, -h                    show help (default: false)
```

### Provider Examples
| Provider | URL                                      |
| -------- | ---------------------------------------- |
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
