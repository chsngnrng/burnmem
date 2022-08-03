This project is based on https://github.com/chaosblade-io/chaosblade-exec-os

It's aim is to adapt memory consumation tool from Chaostools for Windows.

Usage:
chaos_burnmem.exe --mem-percent 0 --reserve 200 --rate 100 --time 600 --swap

When the memory is almost depleted, the utility can fail, since the kernel 
would not always let to allocate, and Goland cannot handle OOM exeption. 
To handle this, there is a watchdog binary burnmem_watchdog.exe, which
will restart chaos_burnmem if it exits before the timer is 0. You can launch it 
with the same options.
