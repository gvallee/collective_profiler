//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package plot

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/notation"
)

const (
	plotScriptPrelude = "set term png size 800,600\nset key outside\nset key right top\n"
)

func sortHostMapKeys(m map[string][]int) []string {
	var array []string
	for k, _ := range m {
		array = append(array, k)
	}
	sort.Strings(array)
	return array
}

func getMax(max int, values map[int]bool, rank int, sendHeatMap map[int]int, recvHeatMap map[int]int, execTimeMap map[int]float64, lateArrivalMap map[int]float64, sendBW float64, recvBW float64) (int, map[int]bool) {
	if max < sendHeatMap[rank] {
		max = sendHeatMap[rank]
	}
	if _, ok := values[sendHeatMap[rank]]; !ok {
		values[sendHeatMap[rank]] = true
	}

	if max < recvHeatMap[rank] {
		max = recvHeatMap[rank]
	}
	if _, ok := values[recvHeatMap[rank]]; !ok {
		values[recvHeatMap[rank]] = true
	}

	v := int(math.Ceil(execTimeMap[rank]))
	if max < v {
		max = v
	}
	if _, ok := values[v]; !ok {
		values[v] = true
	}

	v = int(math.Ceil(lateArrivalMap[rank]))
	if max < v {
		max = v
	}
	if _, ok := values[v]; !ok {
		values[v] = true
	}

	v = int(math.Ceil(sendBW))
	if max < v {
		max = v
	}
	if _, ok := values[v]; !ok {
		values[v] = true
	}

	v = int(math.Ceil(recvBW))
	if max < v {
		max = v
	}
	if _, ok := values[v]; !ok {
		values[v] = true
	}

	return max, values
}

func generateCallDataFiles(dir string, outputDir string, leadRank int, callID int, hostMap map[string][]int, sendHeatMap map[int]int, recvHeatMap map[int]int, execTimeMap map[int]float64, lateArrivalMap map[int]float64) (string, error) {
	hosts := sortHostMapKeys(hostMap)
	maxValue := 0
	numRanks := 0
	values := make(map[int]bool)

	emptyLines := 0
	for _, hostname := range hosts {
		hostFile := filepath.Join(outputDir, hostname+".txt")

		fd, err := os.OpenFile(hostFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return "", err
		}
		defer fd.Close()

		_, err = fd.WriteString("# Rank send_size recv_size exec_time late_time send_bw recv_bw\n")
		if err != nil {
			return "", err
		}

		ranks := hostMap[hostname]
		numRanks += len(ranks)
		sort.Ints(ranks)
		for _, rank := range ranks {
			for i := 0; i < emptyLines; i++ {
				_, err = fd.WriteString("- - - - - - -\n")
				if err != nil {
					return "", err
				}
			}
			sendBW := float64(sendHeatMap[rank]) / execTimeMap[rank]
			recvBW := float64(recvHeatMap[rank]) / execTimeMap[rank]
			maxValue, values = getMax(maxValue, values, rank, sendHeatMap, recvHeatMap, execTimeMap, lateArrivalMap, sendBW, recvBW)
			_, err = fd.WriteString(fmt.Sprintf("%d %d %d %f %f %f %f\n", rank, sendHeatMap[rank], recvHeatMap[rank], execTimeMap[rank], lateArrivalMap[rank], sendBW, recvBW))
			if err != nil {
				return "", err
			}
		}
		emptyLines += len(ranks)
	}

	emptyLines = 0
	idx := 0
	for _, hostname := range hosts {
		hostRankFile := filepath.Join(outputDir, "ranks_map_"+hostname+".txt")

		fd2, err := os.OpenFile(hostRankFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return "", err
		}
		defer fd2.Close()
		for i := 0; i < emptyLines; i++ {
			_, err := fd2.WriteString("0\n")
			if err != nil {
				return "", err
			}
			idx++
		}
		for i := 0; i < len(hostMap[hostname]); i++ {
			_, err := fd2.WriteString(fmt.Sprintf("%d\n", maxValue))
			if err != nil {
				return "", err
			}
			idx++
		}
		for i := idx; i < numRanks; i++ {
			_, err := fd2.WriteString("0\n")
			if err != nil {
				return "", err
			}
			idx++
		}
		emptyLines += len(hostMap[hostname])
	}

	var a []int
	for key, _ := range values {
		a = append(a, key)
	}
	sort.Ints(a)
	gnuplotScript, err := generatePlotScript(outputDir, leadRank, callID, numRanks, maxValue, a, hosts)
	if err != nil {
		return "", err
	}

	return gnuplotScript, nil
}

func generatePlotScript(outputDir string, leadRank int, callID int, numRanks int, maxValue int, values []int, hosts []string) (string, error) {
	plotScriptFile := filepath.Join(outputDir, "profiler_rank"+strconv.Itoa(leadRank)+"_call"+strconv.Itoa(callID)+".gnuplot")
	fd, err := os.OpenFile(plotScriptFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	_, err = fd.WriteString(plotScriptPrelude)
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString(fmt.Sprintf("set output \"profiler_rank%d_call%d.png\"\n\nset pointsize 2\n\n", leadRank, callID))
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString(fmt.Sprintf("set xrange [-1:%d]\n", numRanks))
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString(fmt.Sprintf("set yrange [0:%d]\n", maxValue))
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString("set xtics\n\nset style fill pattern\n\nset style fill solid .1 noborder\nset style line 1 lc rgb 'black' pt 2\nset style line 2 lc rgb 'black' pt 1\nset style line 3 lc rgb 'black' pt 4\nset style line 4 lc rgb 'black' pt 9\nset style line 5 lc rgb 'black' pt 6\n")
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString(fmt.Sprintf("set ytics (%s)\n", notation.IntSliceToString(values)))
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString("\nshow label\n\n")
	if err != nil {
		return "", err
	}
	str := "plot "
	for _, hostname := range hosts {
		str += "\"ranks_map_" + hostname + ".txt\" using 0:1 with boxes title '" + hostname + "', \\\n"
	}
	// Special for the first node
	str += fmt.Sprintf("\"%s.txt\" using 2:xtic(1) with points ls 1 title \"data sent (bytes)\", \\\n", hosts[0])
	str += fmt.Sprintf("\"%s.txt\" using 3 with points ls 2 title \"data received (bytes)\", \\\n", hosts[0])
	str += fmt.Sprintf("\"%s.txt\" using 4 with points ls 3 title \"execution time (s)\", \\\n", hosts[0])
	str += fmt.Sprintf("\"%s.txt\" using 5 with points ls 4 title \"late arrival timing (s)\", \\\n", hosts[0])
	str += fmt.Sprintf("\"%s.txt\" using 6 with points ls 5 title \"bandwidth (bytes/s)\", \\\n", hosts[0])
	for i := 1; i < len(hosts); i++ {
		str += fmt.Sprintf("\"%s.txt\" using 2:xtic(1) with points ls 1 notitle, \\\n", hosts[i])
		str += fmt.Sprintf("\"%s.txt\" using 3 with points ls 2 notitle, \\\n", hosts[i])
		str += fmt.Sprintf("\"%s.txt\" using 4 with points ls 3 notitle, \\\n", hosts[i])
		str += fmt.Sprintf("\"%s.txt\" using 5 with points ls 4 notitle, \\\n", hosts[i])
		str += fmt.Sprintf("\"%s.txt\" using 6 with points ls 5 notitle, \\\n", hosts[i])
	}
	str = strings.TrimRight(str, ", \\\n")
	_, err = fd.WriteString(str)
	if err != nil {
		return "", err
	}

	return plotScriptFile, nil
}

func Create(dir string, outputDir string, leadRank int, callID int, hostMap map[string][]int, sendHeatMap map[int]int, recvHeatMap map[int]int, execTimeMap map[int]float64, lateArrivalMap map[int]float64) error {
	gnuplotScript, err := generateCallDataFiles(dir, outputDir, leadRank, callID, hostMap, sendHeatMap, recvHeatMap, execTimeMap, lateArrivalMap)
	if err != nil {
		return err
	}

	// Run gnuplot
	gnuplotBin, err := exec.LookPath("gnuplot")
	if err != nil {
		return err
	}

	dataPlotScript, err := ioutil.ReadFile(gnuplotScript)
	if err != nil {
		return err
	}

	cmd := exec.Command(gnuplotBin)
	cmd.Dir = outputDir
	cmd.Stdin = bytes.NewBuffer(dataPlotScript)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
