This project is based on https://github.com/chaosblade-io/chaosblade-exec-os

It's aim is to compile memory consumation tool from Chaostools for Windows.
At this moment the changes include removing of features which do not exist in Windows
(cgroups, oom adj, cache mode) and which make sense only in Chaosblade toolset (start/stop)
Further on I hope to make it more neat and convenient to use

Usage:
chaos_burnmem --nohup --mem-percent 0 --reserve 200 --rate 100 --mode ram --include-buffer-cache=false

Exactly all these options should be used, otherwise the tool will try to restart itself using missing /bin/sh
I advise to use --reserve option rather than --mem-percent because the latter seem to work not very reliable on Windows and consume more mem than available
