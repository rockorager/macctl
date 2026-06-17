# macctl

`macctl` is a small `systemctl`-style interface for macOS `launchd`.

It does two simple things:

1. lets you write services and timers as small unit files under XDG-style config
   directories
2. makes setting environment variables for your whole user session less annoying

macOS already has `launchd`, and `macctl` does not replace it. It writes native
LaunchAgent/LaunchDaemon plists and calls `launchctl` for you.

```text
~/.config/macctl/user/*.service  →  ~/Library/LaunchAgents/*.plist
~/.config/macctl/user/*.timer    →  ~/Library/LaunchAgents/*.plist
~/.config/environment.d/*.conf   →  launchctl setenv
```

Instead of writing plist XML or making a one-off LaunchAgent just to set
`EDITOR` or `PATH`, you can do this:

```sh
macctl enable worker.service
macctl start worker
macctl daemon-reload
macctl set-environment EDITOR=nvim
```

## Install

Build from source:

```sh
git clone https://go.rockorager.dev/macctl
cd macctl
mise run build
```

Install the binary:

```sh
mise run install
```

Or with Go directly:

```sh
go install go.rockorager.dev/macctl/cmd/macctl@latest
```

`macctl` is a single binary. It does not install a resident daemon. When you use
user environment files, `macctl daemon-reload` installs a small LaunchAgent so
your environment is applied again at login.

## Usage

### Scopes

User services are the default:

```sh
macctl --user start worker
```

System services use the system launchd domain and require privileges:

```sh
sudo macctl --system start worker
```

### Commands

```sh
macctl start UNIT
macctl stop UNIT
macctl restart UNIT
macctl enable UNIT_OR_PATH
macctl disable UNIT
macctl daemon-reload
macctl set-environment NAME=VALUE
macctl unset-environment NAME
macctl show-environment
macctl import-environment NAME
```

### Services

Create a service file:

```ini
# ~/.config/macctl/user/worker.service
[Unit]
Description=Example worker

[Service]
ExecStart=/Users/tim/bin/worker --config /Users/tim/worker.toml
WorkingDirectory=/Users/tim
Environment=LOG_LEVEL=debug
EnvironmentFile=-/Users/tim/.config/worker.env
Restart=always
RestartSec=5
```

Start it once without enabling it at login:

```sh
macctl start worker.service
```

Enable it at login:

```sh
macctl enable worker.service
```

Restart or inspect it:

```sh
macctl restart worker
```

`macctl` writes a generated LaunchAgent:

```text
~/Library/LaunchAgents/dev.macctl.worker.plist
```

### Timers

Timers use the same paired-file model as systemd. A timer named `backup.timer` runs the matching `backup.service`.

```ini
# ~/.config/macctl/user/backup.timer
[Unit]
Description=Nightly backup

[Timer]
OnCalendar=03:00
```

```ini
# ~/.config/macctl/user/backup.service
[Unit]
Description=Run nightly backup

[Service]
ExecStart=/Users/tim/bin/backup
```

Start it once without enabling it at login:

```sh
macctl start backup.timer
```

Enable it at login:

```sh
macctl enable backup.timer
```

Common timer forms:

```ini
OnCalendar=03:00
OnCalendar=Mon..Fri 09:00
OnCalendar=*-*-01 00:00
OnCalendar=hourly
OnCalendar=daily
OnCalendar=weekly
OnCalendar=monthly
OnUnitActiveSec=30
OnBootSec=10
Unit=other.service
```

`Unit=` selects the service activated by the timer. If omitted, `backup.timer` activates `backup.service`.

`macctl` accepts the standard `systemd.timer` keys where possible. Keys such as `Persistent=`, `AccuracySec=`, and `RandomizedDelaySec=` are parsed but currently have no direct launchd output. Timer schedules compile to launchd `StartCalendarInterval`, `StartInterval`, and `RunAtLoad` settings.

### Environment

Create environment files:

```ini
# ~/.config/environment.d/10-editor.conf
EDITOR=nvim
ROCKORAGER=foo
GOBIN=${HOME}/.local/bin
PATH=${GOBIN}:$PATH
```

Apply them:

```sh
macctl daemon-reload
```

Check a value:

```sh
launchctl getenv ROCKORAGER
```

Set or import values imperatively:

```sh
macctl set-environment EDITOR=nvim
macctl import-environment SSH_AUTH_SOCK
macctl show-environment
```

### Paths

User config:

```text
$XDG_CONFIG_HOME/macctl/user/*.service
$XDG_CONFIG_HOME/macctl/user/*.timer
$XDG_CONFIG_HOME/environment.d/*.conf
```

If `XDG_CONFIG_HOME` is unset, it defaults to `~/.config`.

System config:

```text
/etc/xdg/macctl/system/*.service
/etc/xdg/macctl/system/*.timer
/etc/environment.d/*.conf
```

Generated launchd plists:

```text
~/Library/LaunchAgents/dev.macctl.*.plist
/Library/LaunchDaemons/dev.macctl.*.plist
```
