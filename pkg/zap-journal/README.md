# zap-journal

This is a extension of zap which implement systemd-journal `zapcore.Encoder` and
`zapcore.WriteSyncer`.

For code example, check ./example/example.go.

## Note

1. The default logger setup produce a named logger with the executable file name
   get from `os.Executable()`. This behavior is different from zap.

2. The key passed to zap will be convert to UPPERCASE. As journald seems refuse
   to record lowercase fields.

3. `journalctl` will not show any custom fields by default. And it not support
   any format customization, but output it as json stream. If you need something
   to format that json stream into a human readable log, check [journalfmt][1].

[1]: https://github.com/black-desk/journalfmt
