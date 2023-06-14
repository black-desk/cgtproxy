# cgtproxy

cgtproxy is a transparent proxy **RULE** manager written in go.
It will automatically update your nft ruleset according to your configuration,
to archive per-app transparent proxy settings.

## How it works

Netfliter can be configured to filter network traffic [by cgroup],
as well as redirect some traffic to a [TPROXY] server.

[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html
[by cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html

Systemd has a work-in-progress XDG integration [documentation] suggest that
XDG applications should be launched in a systemd managed unit.

[documentation]: https://systemd.io/DESKTOP_ENVIRONMENTS

For example, telegram might be launched at some cgroup like
`/user.slice/user-1000.slice/user@1000.service/app.slice/app-flatpak-org.telegram.desktop@12345.service`

That means the cgroup path for the application has a pattern,
which we can match by a regex expression.

This program will listening cgroupfs change with inotify.
And update the nftable rules when new cgroup hierarchy created,
according to your configuration.

## Configuration

[example](./misc/config/example_without_repeater.yaml)

## Tips

If you want a temporary shell without any transparent proxy,
you can write rules like this:

```yaml
- match: \/system\.slice\/system-clash\d+\.slice/.*
  direct: true
- match: \/user\.slice\/user-\d+\.slice/user@\d+\.service\/direct\.slice\/.*
  direct: true
- match: \/.*
  tproxy: clash
```

Then you can just start a new shell with this command.

```bash
systemd-run --user --shell --slice direct
```

The command above start the new shell in a cgroup like
`/user.slice/user-1000.slice/user@1000.service/direct.slice/run-u22.service`,
which match the regex in your configuration.

Then cgtproxy will produce nft rules to
make that `run-u22.service` get rid of transparent proxy.

## Differences between cgproxy

This project is inspired by [cgproxy](https://github.com/springzfx/cgproxy).

But there are some differences between cgproxy and cgtproxy:

- cgproxy using iptables, but cgtproxy use nftables.

  <https://wiki.nftables.org/wiki-nftables/index.php/Main_differences_with_iptables>

- cgproxy using REDIR **AND** TPROXY only, but cgtproxy use only TPROXY.

- cgproxy can only working with exsiting cgroup,
  but cgtproxy can update nftables rules dynamically for new cgroups.

- cgproxy use BPF, but cgtproxy not;

  cgproxy handle per-app proxy by using BPF to trace the execve syscall.
  If the new executable file of that process matched some pattern,
  cgproxy daemon will put that process into a special hierarchy `proxy.slice`.

  This weird behavior make process escape from the user-level hierarchy,
  which means the systemd resource control settings will not take any effect.

- cgproxy require at least CAP_NETWORK_ADMIN and CAP_BPF,
  but cgtproxy require only CAP_NETWORK_ADMIN.

  Check the [systemd service file] for details.

- Programs never get moved from original cgroup;

  cgproxy daemon will move processes
  with certain executable path into special cgroup.
  cgtproxy will never do that.

[systemd service file]: ./misc/systemd/cgtproxy.service

## TODO

- [ ] optional cgroup monitor implementation listening on D-Bus
      instead of filesystem;
- [ ] DNS hijack for fake-ip;
- [ ] builtin TPROXY server.

## Develop

Check this documentation [here](docs/development.md)
