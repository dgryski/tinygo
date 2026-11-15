# Run gnuplot plot.gp to draw the charts from the data files in this directory.
set terminal pngcairo size 1800,1100 font "Helvetica,13"

# Okabe-Ito palette
c_fnv = "#0072B2"; c_ldb = "#009E73"; c_t32 = "#E69F00"
c_t64 = "#CC79A7"; c_s13 = "#D55E00"; c_s24 = "#56B4E9"; c_fin = "#999999"

boards = "uno_r4_m4 pico2w_m33 xiao_samd21_m0plus pico1_rp2040 microbit_nrf51"
titles = "'Arduino Uno R4 WiFi (Cortex-M4, 48 MHz)' 'Pico 2 W (Cortex-M33, 150 MHz)' 'XIAO SAMD21 (Cortex-M0+, 48 MHz)' 'Pico 1 (RP2040, Cortex-M0+, 125 MHz)' 'micro:bit v1 (nRF51, Cortex-M0, 16 MHz)'"

# 1. Cycles per hash by key class (log scale)
set output "hash_cycles.png"
set multiplot layout 2,3 title "Hash cost on real hardware: net cycles per call (lower is better, log scale)" font ",16"
set style data histogram
set style histogram clustered gap 1
set style fill solid 0.9 border -1
set boxwidth 0.9
set logscale y
set yrange [20:8000]
set ytics (30,50,100,200,500,1000,2000,5000)
set grid ytics
set xtics rotate by -30 font ",11"
set ylabel "net cycles / call"
do for [i=1:5] {
  set title word(titles,i) font ",13"
  if (i==1) { set key top left font ",11" } else { unset key }
  plot "cycles_".word(boards,i).".dat" using 3:xtic(1) title "fnv" lc rgb c_fnv, \
       "" using 4 title "leveldb" lc rgb c_ldb, \
       "" using 5 title "tsip32" lc rgb c_t32, \
       "" using 6 title "tsip64" lc rgb c_t64, \
       "" using 7 title "sip13" lc rgb c_s13, \
       "" using 8 title "sip24" lc rgb c_s24
}
unset multiplot

# 2. Cost relative to leveldb (log2 scale, 1.0 = leveldb); fnv ratio only,
#    plus tsip32 and sip13
reset
set terminal pngcairo size 1800,1100 font "Helvetica,13"
set output "relative_to_leveldb.png"
c_fnv = "#0072B2"; c_t32 = "#E69F00"; c_s13 = "#D55E00"
set multiplot layout 2,3 title "Hash cost relative to leveldb on real hardware (1.0 = leveldb; below 1 is faster)" font ",16"
set style data histogram
set style histogram clustered gap 1
set style fill solid 0.9 border -1
set boxwidth 0.9
set logscale y 2
set yrange [0.5:16]
set ytics (0.5,1,2,4,8,16)
set grid ytics
set xtics rotate by -30 font ",11"
set ylabel "cycles relative to leveldb"
do for [i=1:5] {
  set title word(titles,i) font ",13"
  if (i==1) { set key top left font ",11" } else { unset key }
  set arrow 1 from graph 0, first 1 to graph 1, first 1 nohead lw 2 lc rgb "#444444" dt 2 front
  plot "ratio_".word(boards,i).".dat" using 3:xtic(1) title "fnv" lc rgb c_fnv, \
       "" using 4 title "tsip32" lc rgb c_t32, \
       "" using 5 title "sip13" lc rgb c_s13
  unset arrow 1
}
unset multiplot

# 3. Cycles vs key length (log-log), fnv vs leveldb vs tsip32
reset
set terminal pngcairo size 1800,1100 font "Helvetica,13"
set output "cycles_vs_keylen.png"
c_fnv = "#0072B2"; c_ldb = "#009E73"; c_t32 = "#E69F00"; c_fin = "#999999"
set multiplot layout 2,3 title "fnv vs leveldb vs tsip32: cycles per call vs key length (log-log)" font ",16"
set logscale xy
set xrange [3:170]
set yrange [20:3000]
set ytics (30,50,100,200,500,1000,2000)
set xtics (4,8,14,39,128)
set grid
set xlabel "key length (bytes)"
set ylabel "net cycles / call"
do for [i=1:5] {
  set title word(titles,i) font ",13"
  if (i==1) { set key top left font ",11" } else { unset key }
  f = "cycles_".word(boards,i).".dat"
  plot f using 2:3 title "fnv" with linespoints lw 2 pt 7 lc rgb c_fnv, \
       f using 2:9 title "fnv+finalizer" with linespoints lw 2 pt 6 dt 2 lc rgb c_fin, \
       f using 2:4 title "leveldb" with linespoints lw 2 pt 5 lc rgb c_ldb, \
       f using 2:5 title "tsip32" with linespoints lw 2 pt 9 lc rgb c_t32
}
unset multiplot

# 4. AVR (Uno R3), cycles per call
reset
set terminal pngcairo size 1100,640 font "Helvetica,13"
set output "avr_cycles.png"
c_fnv = "#0072B2"; c_ldb = "#009E73"; c_t32 = "#E69F00"; c_s13 = "#D55E00"; c_fin = "#999999"
set title "AVR (ATmega328p, 16 MHz): net cycles per call, log scale" font ",15"
set style data histogram
set style histogram clustered gap 1
set style fill solid 0.9 border -1
set boxwidth 0.9
set logscale y
set yrange [300:100000]
set ytics (500,1000,2000,5000,10000,20000,50000)
set grid ytics
set ylabel "net cycles / call"
set xlabel "key length"
set key top left
plot "cycles_avr_uno_r3.dat" using 3:xtic(1) title "fnv" lc rgb c_fnv, \
     "" using 4 title "fnv+finalizer" lc rgb c_fin, \
     "" using 5 title "leveldb" lc rgb c_ldb, \
     "" using 6 title "tsip32" lc rgb c_t32, \
     "" using 7 title "sip13" lc rgb c_s13

# 5. Code size per hash function
reset
set terminal pngcairo size 1100,640 font "Helvetica,13"
set output "codesize.png"
c_fnv = "#0072B2"; c_ldb = "#009E73"; c_t32 = "#E69F00"
c_t64 = "#CC79A7"; c_s13 = "#D55E00"; c_s24 = "#56B4E9"
set title "Code size of each hash function (bytes, from linked binaries)" font ",15"
set style data histogram
set style histogram clustered gap 1
set style fill solid 0.9 border -1
set boxwidth 0.9
set logscale y
set yrange [20:3000]
set ytics (30,50,100,200,500,1000,2000)
set grid ytics
set ylabel "bytes of code"
set key top left
plot "codesize.dat" using 2:xtic(1) title "fnv" lc rgb c_fnv, \
     "" using 3 title "leveldb" lc rgb c_ldb, \
     "" using 4 title "tsip32" lc rgb c_t32, \
     "" using 5 title "tsip64" lc rgb c_t64, \
     "" using 6 title "sip13" lc rgb c_s13, \
     "" using 7 title "sip24" lc rgb c_s24
