This project is based on https://github.com/chaosblade-io/chaosblade-exec-os

It's aim is to adapt memory consumation tool from Chaostools for Windows.

Usage:
chaos_burnmem --mem-percent 0 --reserve 200 --rate 100 --time 600

I advise to use --reserve option rather than --mem-percent because the latter seem to work not very reliable on Windows Server and consume more mem than available