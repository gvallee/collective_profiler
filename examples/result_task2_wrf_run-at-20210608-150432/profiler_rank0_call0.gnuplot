set term png size 3200,2400
set key out vert
set key right
set output "profiler_rank0_call0.png"

set pointsize 2

set xrange [-1:160]
set yrange [0:1000]
set xtics

set style fill pattern

set style fill solid .1 noborder
set style line 1 lc rgb 'black' pt 2
set style line 2 lc rgb 'blue' pt 1
set style line 3 lc rgb 'red' pt 4
set style line 4 lc rgb 'pink' pt 9
set style line 5 lc rgb 'green' pt 6

show label

plot "ranks_map_node-002.txt" using 0:1 with boxes title 'node-002', \
"ranks_map_node-012.txt" using 0:1 with boxes title 'node-012', \
"ranks_map_node-017.txt" using 0:1 with boxes title 'node-017', \
"ranks_map_node-031.txt" using 0:1 with boxes title 'node-031', \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-002.txt.txt" using 2:xtic(1) with points ls 1 title "data sent (B)", \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-002.txt.txt" using 3 with points ls 2 title "data received (B)", \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-002.txt.txt" using 4 with points ls 3 title "execution time (milliseconds)", \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-002.txt.txt" using 5 with points ls 4 title "late arrival timing (milliseconds)", \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-002.txt.txt" using 6 with points ls 5 title "bandwidth (B/s)", \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-012.txt.txt" using 2:xtic(1) with points ls 1 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-012.txt.txt" using 3 with points ls 2 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-012.txt.txt" using 4 with points ls 3 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-012.txt.txt" using 5 with points ls 4 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-012.txt.txt" using 6 with points ls 5 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-017.txt.txt" using 2:xtic(1) with points ls 1 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-017.txt.txt" using 3 with points ls 2 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-017.txt.txt" using 4 with points ls 3 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-017.txt.txt" using 5 with points ls 4 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-017.txt.txt" using 6 with points ls 5 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-031.txt.txt" using 2:xtic(1) with points ls 1 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-031.txt.txt" using 3 with points ls 2 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-031.txt.txt" using 4 with points ls 3 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-031.txt.txt" using 5 with points ls 4 notitle, \
"/Volumes/DataCorrupted/project/isc/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/data_rank0_call0_node-031.txt.txt" using 6 with points ls 5 notitle