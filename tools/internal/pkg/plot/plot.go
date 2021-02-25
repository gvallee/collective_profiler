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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/scale"
	"github.com/gvallee/go_util/pkg/util"
)

const (
	plotScriptPrelude = "set term png size 3200,2400\nset key out vert\nset key right\n"
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

// fixme: too similar to generateCallDataFiles
func generateAvgsDataFiles(dir string, outputDir string, hostMap map[string][]int, avgSendHeatMap map[int]int, avgRecvHeatMap map[int]int, avgExecTimeMap map[int]float64, avgLateArrivalTimeMap map[int]float64) (string, error) {
	if avgSendHeatMap == nil {
		return "", fmt.Errorf("avgSendHeatMap is undefined")
	}
	if avgRecvHeatMap == nil {
		return "", fmt.Errorf("avgRecvHeatMap is undefined")
	}
	if avgExecTimeMap == nil {
		return "", fmt.Errorf("avgExecTimeMap is undefined")
	}
	if avgLateArrivalTimeMap == nil {
		return "", fmt.Errorf("avgLateArrivalTimeMap is undefined")
	}

	if len(avgSendHeatMap) == 0 {
		return "", fmt.Errorf("avgSendHeatMap is empty")
	}
	if len(avgRecvHeatMap) == 0 {
		return "", fmt.Errorf("avgRecvHeatMap is empty")
	}
	if len(avgExecTimeMap) == 0 {
		return "", fmt.Errorf("avgExecTimeMap is empty")
	}
	if len(avgLateArrivalTimeMap) == 0 {
		return "", fmt.Errorf("avgLateArrivalTimeMap is empty")
	}

	hosts := sortHostMapKeys(hostMap)
	maxValue := 1000 // We automatically scale the data, the max is always 1000
	numRanks := 0
	values := make(map[int]bool)
	sendRankBW := make(map[int]float64)
	recvRankBW := make(map[int]float64)
	scaledSendRankBW := make(map[int]float64)
	scaledRecvRankBW := make(map[int]float64)

	avgSendHeatMapUnit, avgSendScaledHeatMap, err := scale.MapInts("B", avgSendHeatMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapInts() on avgSendHeatMap failed(): %s", err)
	}
	avgRecvHeatMapUnit, avgRecvScaledHeatMap, err := scale.MapInts("B", avgRecvHeatMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapInts() on avgRecvHeatMap failed(): %s", err)
	}
	avgExecTimeMapUnit, avgExecScaledTimeMap, err := scale.MapFloat64s("seconds", avgExecTimeMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapFloat64s() on avgExecTimeMap failed(): %s", err)
	}
	// debug
	if avgLateArrivalTimeMap == nil {
		return "", fmt.Errorf("TOTO avgLateArrivalTimeMap is undefined")
	}
	avgLateArrivalTimeMapUnit, avgLateArrivalScaledTimeMap, err := scale.MapFloat64s("seconds", avgLateArrivalTimeMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapFloat64s() on avgLateArrivalTimeMap failed(): %s", err)
	}

	// fixme: atm we assume that all BW data is homogeneous so once we figure out a scale, it
	// is the same scale all the time. It might not be true so we really need to figure out the
	// scale based on sendHeatMapUnit and recvHeatMapUnit and force it to be used later when
	// calculating the bandwidth
	sBWUnit := ""
	rBWUnit := ""

	emptyLines := 0
	for _, hostname := range hosts {
		hostFile := filepath.Join(outputDir, hostname+"_avgs.txt")

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
		for i := 0; i < emptyLines; i++ {
			_, err = fd.WriteString("- - - - - - -\n")
			if err != nil {
				return "", err
			}
		}
		for _, rank := range ranks {
			sendRankBW[rank] = float64(avgSendHeatMap[rank]) / avgExecTimeMap[rank]
			recvRankBW[rank] = float64(avgRecvHeatMap[rank]) / avgExecTimeMap[rank]
			var scaledSendRankBWUnit string
			var scaledRecvRankBWUnit string
			scaledSendRankBWUnit, scaledSendBW, err := scale.Float64s("B/s", []float64{sendRankBW[rank]})
			if err != nil {
				return "", err
			}
			scaledSendRankBW[rank] = scaledSendBW[0]
			scaledRecvRankBWUnit, scaledRecvBW, err := scale.Float64s("B/s", []float64{recvRankBW[rank]})
			if err != nil {
				return "", err
			}
			scaledRecvRankBW[rank] = scaledRecvBW[0]
			if sBWUnit != "" && sBWUnit != scaledSendRankBWUnit {
				return "", fmt.Errorf("detected different scales for BW data")
			}
			if rBWUnit != "" && rBWUnit != scaledRecvRankBWUnit {
				return "", fmt.Errorf("detected different scales for BW data")
			}
			if sBWUnit == "" {
				sBWUnit = scaledSendRankBWUnit
			}
			if rBWUnit == "" {
				rBWUnit = scaledRecvRankBWUnit
			}

			_, values = getMax(maxValue, values, rank, avgSendScaledHeatMap, avgRecvScaledHeatMap, avgExecScaledTimeMap, avgLateArrivalScaledTimeMap, sendRankBW[rank], recvRankBW[rank])
			_, err = fd.WriteString(fmt.Sprintf("%d %d %d %f %f %f %f\n", rank, avgSendScaledHeatMap[rank], avgRecvScaledHeatMap[rank], avgExecScaledTimeMap[rank], avgLateArrivalScaledTimeMap[rank], sendRankBW[0], recvRankBW[1]))
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
	gnuplotScript, err := generateGlobalPlotScript(outputDir, numRanks, maxValue, a, hosts, avgSendHeatMapUnit, avgRecvHeatMapUnit, avgExecTimeMapUnit, avgLateArrivalTimeMapUnit, sBWUnit, rBWUnit)
	if err != nil {
		return "", err
	}

	return gnuplotScript, nil
}

func generateCallDataFiles(dir string, outputDir string, leadRank int, callID int, hostMap map[string][]int, sendHeatMap map[int]int, recvHeatMap map[int]int, execTimeMap map[int]float64, lateArrivalMap map[int]float64) (string, string, error) {
	if sendHeatMap == nil {
		return "", "", fmt.Errorf("avgSendHeatMap is undefined")
	}
	if recvHeatMap == nil {
		return "", "", fmt.Errorf("avgRecvHeatMap is undefined")
	}
	if execTimeMap == nil {
		return "", "", fmt.Errorf("avgExecTimeMap is undefined")
	}
	if lateArrivalMap == nil {
		return "", "", fmt.Errorf("avgLateArrivalTimeMap is undefined")
	}

	hosts := sortHostMapKeys(hostMap)
	maxValue := 1000 // We scale the data the maximum is always 1000
	numRanks := 0
	values := make(map[int]bool)
	sendRankBW := make(map[int]float64)
	recvRankBW := make(map[int]float64)

	sendHeatMapUnit, sendScaledHeatMap, err := scale.MapInts("B", sendHeatMap)
	if err != nil {
		return "", "", err
	}
	recvHeatMapUnit, recvScaledHeatMap, err := scale.MapInts("B", recvHeatMap)
	if err != nil {
		return "", "", err
	}
	execTimeMapUnit, execScaledTimeMap, err := scale.MapFloat64s("seconds", execTimeMap)
	if err != nil {
		return "", "", err
	}
	lateArrivalTimeMapUnit, lateArrivalScaledTimeMap, err := scale.MapFloat64s("seconds", lateArrivalMap)
	if err != nil {
		return "", "", err
	}

	// fixme: atm we assume that all BW data is homogeneous so once we figure out a scale, it
	// is the same scale all the time. It might not be true so we really need to figure out the
	// scale based on sendHeatMapUnit and recvHeatMapUnit and force it to be used later when
	// calculating the bandwidth
	sBWUnit := ""
	rBWUnit := ""

	emptyLines := 0
	for _, hostname := range hosts {
		hostFile := filepath.Join(outputDir, hostname+".txt")

		fd, err := os.OpenFile(hostFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return "", "", err
		}
		defer fd.Close()

		_, err = fd.WriteString("# Rank send_size recv_size exec_time late_time send_bw recv_bw\n")
		if err != nil {
			return "", "", err
		}

		ranks := hostMap[hostname]
		numRanks += len(ranks)
		sort.Ints(ranks)
		for i := 0; i < emptyLines; i++ {
			_, err = fd.WriteString("- - - - - - -\n")
			if err != nil {
				return "", "", err
			}
		}
		for _, rank := range ranks {
			sendRankBW[rank] = float64(sendHeatMap[rank]) / execTimeMap[rank]
			recvRankBW[rank] = float64(recvHeatMap[rank]) / execTimeMap[rank]
			scaledSendRankBWUnit, scaledSendRankBW, err := scale.MapFloat64s("B/s", sendRankBW)
			if err != nil {
				return "", "", err
			}
			scaledRecvRankBWUnit, scaledRecvRankBW, err := scale.MapFloat64s("B/s", recvRankBW)
			if err != nil {
				return "", "", err
			}
			if sBWUnit != "" && sBWUnit != scaledSendRankBWUnit {
				return "", "", fmt.Errorf("detected different scales for BW data")
			}
			if rBWUnit != "" && rBWUnit != scaledRecvRankBWUnit {
				return "", "", fmt.Errorf("detected different scales for BW data")
			}
			if sBWUnit != "" && sBWUnit != scaledSendRankBWUnit {
				return "", "", fmt.Errorf("detected different scales for BW data")
			}
			if rBWUnit != "" && rBWUnit != scaledRecvRankBWUnit {
				return "", "", fmt.Errorf("detected different scales for BW data")
			}
			if sBWUnit == "" {
				sBWUnit = scaledSendRankBWUnit
			}
			if rBWUnit == "" {
				rBWUnit = scaledRecvRankBWUnit
			}

			_, values = getMax(maxValue, values, rank, sendScaledHeatMap, recvScaledHeatMap, execScaledTimeMap, lateArrivalScaledTimeMap, scaledSendRankBW[rank], scaledRecvRankBW[rank])
			_, err = fd.WriteString(fmt.Sprintf("%d %d %d %f %f %f %f\n", rank, sendScaledHeatMap[rank], recvScaledHeatMap[rank], execScaledTimeMap[rank], lateArrivalScaledTimeMap[rank], scaledSendRankBW[rank], scaledRecvRankBW[rank]))
			if err != nil {
				return "", "", err
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
			return "", "", err
		}
		defer fd2.Close()
		for i := 0; i < emptyLines; i++ {
			_, err := fd2.WriteString("0\n")
			if err != nil {
				return "", "", err
			}
			idx++
		}
		for i := 0; i < len(hostMap[hostname]); i++ {
			_, err := fd2.WriteString(fmt.Sprintf("%d\n", maxValue))
			if err != nil {
				return "", "", err
			}
			idx++
		}
		for i := idx; i < numRanks; i++ {
			_, err := fd2.WriteString("0\n")
			if err != nil {
				return "", "", err
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

	pngFile, gnuplotScript, err := generateCallPlotScript(outputDir, leadRank, callID, numRanks, maxValue, a, hosts, sendHeatMapUnit, recvHeatMapUnit, execTimeMapUnit, lateArrivalTimeMapUnit, sBWUnit, rBWUnit)
	if err != nil {
		return "", "", fmt.Errorf("plot.generateCallPlotScript() failed: %s", err)
	}

	return pngFile, gnuplotScript, nil
}

// fixme: values is not currently used, by removing it, the code would become much simpler.
func write(fd *os.File, numRanks int, maxValue int, values []int, hosts []string, sendUnit string, recvUnit string, execTimeUnit string, lateArrivalTimeUnit string, sendBWUnit string, recvBWUnit string) error {
	if len(hosts) == 0 {
		return fmt.Errorf("empty list of hosts")
	}
	_, err := fd.WriteString(fmt.Sprintf("set xrange [-1:%d]\n", numRanks))
	if err != nil {
		return err
	}
	_, err = fd.WriteString(fmt.Sprintf("set yrange [0:1000]\n"))
	if err != nil {
		return err
	}
	_, err = fd.WriteString("set xtics\n\nset style fill pattern\n\nset style fill solid .1 noborder\nset style line 1 lc rgb 'black' pt 2\nset style line 2 lc rgb 'blue' pt 1\nset style line 3 lc rgb 'red' pt 4\nset style line 4 lc rgb 'pink' pt 9\nset style line 5 lc rgb 'green' pt 6\n")
	if err != nil {
		return err
	}
	/*
		_, err = fd.WriteString(fmt.Sprintf("set ytics (%s)\n", notation.IntSliceToString(values)))
		if err != nil {
			return err
		}
	*/
	_, err = fd.WriteString("\nshow label\n\n")
	if err != nil {
		return err
	}
	str := "plot "
	for _, hostname := range hosts {
		str += "\"ranks_map_" + hostname + ".txt\" using 0:1 with boxes title '" + hostname + "', \\\n"
	}

	// Special for the first node
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 2:xtic(1) with points ls 1 title \"data sent (%s)\", \\\n", hosts[0], sendUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 3 with points ls 2 title \"data received (%s)\", \\\n", hosts[0], recvUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 4 with points ls 3 title \"execution time (%s)\", \\\n", hosts[0], execTimeUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 5 with points ls 4 title \"late arrival timing (%s)\", \\\n", hosts[0], lateArrivalTimeUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 6 with points ls 5 title \"bandwidth (%s)\", \\\n", hosts[0], sendBWUnit))
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
		return err
	}

	return nil
}

func getPlotFilename(leadRank int, callID int) string {
	return fmt.Sprintf("profiler_rank%d_call%d.png", leadRank, callID)
}

func generateCallPlotScript(outputDir string, leadRank int, callID int, numRanks int, maxValue int, values []int, hosts []string, sendUnit string, recvUnit string, execTimeUnit string, lateTimeUnit string, sendBWUnit string, recvBWUnit string) (string, string, error) {
	plotScriptFile := filepath.Join(outputDir, "profiler_rank"+strconv.Itoa(leadRank)+"_call"+strconv.Itoa(callID)+".gnuplot")
	fd, err := os.OpenFile(plotScriptFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", "", err
	}
	defer fd.Close()

	_, err = fd.WriteString(plotScriptPrelude)
	if err != nil {
		return "", "", err
	}
	targetPlotFile := getPlotFilename(leadRank, callID)
	_, err = fd.WriteString(fmt.Sprintf("set output \"%s\"\n\nset pointsize 2\n\n", targetPlotFile))
	if err != nil {
		return "", "", err
	}

	err = write(fd, numRanks, maxValue, values, hosts, sendUnit, recvUnit, execTimeUnit, lateTimeUnit, sendBWUnit, recvBWUnit)
	if err != nil {
		return "", "", err
	}

	return targetPlotFile, plotScriptFile, nil
}

func generateGlobalPlotScript(outputDir string, numRanks int, maxValue int, values []int, hosts []string, sendUnit string, recvUnit string, execTimeUnit string, lateTimeUnit string, sendBWUnit string, recvBWUnit string) (string, error) {
	plotScriptFile := filepath.Join(outputDir, "profiler_avgs.gnuplot")
	fd, err := os.OpenFile(plotScriptFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	_, err = fd.WriteString(plotScriptPrelude)
	if err != nil {
		return "", err
	}
	_, err = fd.WriteString("set output \"profiler_avgs.png\"\n\nset pointsize 2\n\n")
	if err != nil {
		return "", err
	}
	err = write(fd, numRanks, maxValue, values, hosts, sendUnit, recvUnit, execTimeUnit, lateTimeUnit, sendBWUnit, recvBWUnit)
	if err != nil {
		return "", err
	}

	return plotScriptFile, nil
}

func runGnuplot(gnuplotScript string, outputDir string) error {
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

// CallFilesExist checks if all the expected files for a specific collective call already exist
func CallFilesExist(outputDir string, leadRank int, callID int) bool {
	return util.PathExists(filepath.Join(outputDir, getPlotFilename(leadRank, callID)))
}

// CallData plots the data related to a specific collective call
func CallData(dir string, outputDir string, leadRank int, callID int, hostMap map[string][]int, sendHeatMap map[int]int, recvHeatMap map[int]int, execTimeMap map[int]float64, lateArrivalMap map[int]float64) (string, error) {
	if len(hostMap) == 0 {
		return "", fmt.Errorf("empty list of hosts")
	}
	if len(sendHeatMap) == 0 {
		return "", fmt.Errorf("sendHeatMap is empty")
	}
	if len(recvHeatMap) == 0 {
		return "", fmt.Errorf("recvHeatMap is empty")
	}
	if len(execTimeMap) == 0 {
		return "", fmt.Errorf("execTimeMap is empty")
	}
	if len(lateArrivalMap) == 0 {
		return "", fmt.Errorf("lateArrivalMap")
	}

	pngFile, gnuplotScript, err := generateCallDataFiles(dir, outputDir, leadRank, callID, hostMap, sendHeatMap, recvHeatMap, execTimeMap, lateArrivalMap)
	if err != nil {
		return "", fmt.Errorf("generateCallDataFiles() failed: %s", err)
	}

	return pngFile, runGnuplot(gnuplotScript, outputDir)
}

// Avgs plots the average statistics gathered during the post-mortem analysis
func Avgs(dir string, outputDir string, numRanks int, hostMap map[string][]int, avgSendHeatMap map[int]int, avgRecvHeatMap map[int]int, avgExecTimeMap map[int]float64, avgLateArrivalTimeMap map[int]float64) error {
	gnuplotScript, err := generateAvgsDataFiles(dir, outputDir, hostMap, avgSendHeatMap, avgRecvHeatMap, avgExecTimeMap, avgLateArrivalTimeMap)
	if err != nil {
		return fmt.Errorf("generateAvgsDataFiles() failed: %s", err)
	}

	return runGnuplot(gnuplotScript, outputDir)
}
