Map and hash benchmarks.

Map benchmarks use random ints, short words and HTTP header names as keys.
Pick the runtime hash with a build tag.

    tinygo test -tags=runtime_memhash_leveldb -bench Map ./tests/mapbench

board times the hash functions directly on Cortex-M and ESP32-S3 boards.

    tinygo flash -monitor -target=pico ./tests/mapbench/board

avr does the same for ATmega328p boards. Build it with `-tags=once` to run a
single pass in simavr.

    tinygo flash -monitor -baudrate=9600 -target=arduino ./tests/mapbench/avr

results has the measured data and the gnuplot script that draws the charts.
