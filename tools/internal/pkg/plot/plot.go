//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
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
	for k := range m {
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

type plotData struct {
	outputDir                   string
	hostMap                     map[string][]int
	values                      map[int]bool
	sendRankBW                  map[int]float64
	recvRankBW                  map[int]float64
	scaledSendRankBW            map[int]float64
	scaledRecvRankBW            map[int]float64
	avgSendScaledHeatMap        map[int]int
	avgRecvScaledHeatMap        map[int]int
	avgExecScaledTimeMap        map[int]float64
	avgLateArrivalScaledTimeMap map[int]float64
	sendScaledHeatMap           map[int]int
	recvScaledHeatMap           map[int]int
	execScaledTimeMap           map[int]float64
	lateArrivalScaledTimeMap    map[int]float64
	emptyLines                  int
	avgSendHeatMap              map[int]int
	avgRecvHeatMap              map[int]int
	avgExecTimeMap              map[int]float64
	avgLateArrivalTimeMap       map[int]float64
	sendHeatMap                 map[int]int
	recvHeatMap                 map[int]int
	execTimeMap                 map[int]float64
	lateArrivalTimeMap          map[int]float64
	maxValue                    int
	sBWUnit                     string
	rBWUnit                     string
	avgSendHeatMapUnit          string
	avgRecvHeatMapUnit          string
	avgExecTimeMapUnit          string
	avgLateArrivalTimeMapUnit   string
	sendHeatMapUnit             string
	recvHeatMapUnit             string
	execTimeMapUnit             string
	lateArrivalTimeMapUnit      string
	numRanks                    int
}

func (d *plotData) generateRanksMap(idx int, hostname string) (int, int, error) {
	hostRankFile := filepath.Join(d.outputDir, "ranks_map_"+hostname+".txt")

	fd2, err := os.OpenFile(hostRankFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return 0, 0, err
	}
	defer fd2.Close()

	for i := 0; i < d.emptyLines; i++ {
		_, err := fd2.WriteString("0\n")
		if err != nil {
			return 0, 0, err
		}
		idx++
	}
	for i := 0; i < len(d.hostMap[hostname]); i++ {
		_, err := fd2.WriteString(fmt.Sprintf("%d\n", d.maxValue))
		if err != nil {
			return 0, 0, err
		}
		idx++
	}
	for i := idx; i < d.numRanks; i++ {
		_, err := fd2.WriteString("0\n")
		if err != nil {
			return 0, 0, err
		}
		idx++
	}
	return len(d.hostMap[hostname]), idx, err
}

func (d *plotData) generateHostCallData(hostname string, idx int) (int, error) {
	hostRankFile := filepath.Join(d.outputDir, "ranks_map_"+hostname+".txt")

	fd2, err := os.OpenFile(hostRankFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return idx, err
	}
	defer fd2.Close()

	for i := 0; i < d.emptyLines; i++ {
		_, err := fd2.WriteString("0\n")
		if err != nil {
			return idx, err
		}
		idx++
	}
	for i := 0; i < len(d.hostMap[hostname]); i++ {
		_, err := fd2.WriteString(fmt.Sprintf("%d\n", d.maxValue))
		if err != nil {
			return idx, err
		}
		idx++
	}
	for i := idx; i < d.numRanks; i++ {
		_, err := fd2.WriteString("0\n")
		if err != nil {
			return idx, err
		}
		idx++
	}
	return idx, nil
}

func (d *plotData) generateCallsAvgs(hostname string, leadRank int, callID int) error {
	dataFile := getPlotDataFilePath(d.outputDir, leadRank, callID, hostname)

	fd, err := os.OpenFile(dataFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.WriteString("# Rank send_size recv_size exec_time late_time send_bw recv_bw\n")
	if err != nil {
		return err
	}

	ranks := d.hostMap[hostname]
	d.numRanks += len(ranks)
	sort.Ints(ranks)
	for i := 0; i < d.emptyLines; i++ {
		_, err = fd.WriteString("- - - - - - -\n")
		if err != nil {
			return err
		}
	}
	for _, rank := range ranks {
		d.sendRankBW[rank] = float64(d.sendHeatMap[rank]) / d.execTimeMap[rank]
		d.recvRankBW[rank] = float64(d.recvHeatMap[rank]) / d.execTimeMap[rank]

		// If the average is different from 0, we try to scale it and hope that the scale
		// will match what we already have for other values. If not, we fail, we have no
		// mechanism to put various data to the same scale at the moment.
		// So, before starting to do some calculation, we assume the following default values
		// which are used when the average is equal to 0:
		// - the scaled BW is equal to non-scaled BW
		// - the unit is the one previous detected (by default the average is assumed to be
		// 	 equal to 0 so it does not matter)
		// Also note that if the rank is not in the communicator, the value is set to 'NaN'

		scaledSendRankBW := d.sendRankBW
		if d.sendRankBW[rank] != 0 && !math.IsNaN(d.sendRankBW[rank]) {
			scaledSendRankBWUnit := d.sBWUnit
			scaledSendRankBWUnit, scaledSendRankBW, err = scale.MapFloat64s("B/s", d.sendRankBW)
			if err != nil {
				return err
			}
			if d.sBWUnit != "" && d.sBWUnit != scaledSendRankBWUnit {
				return fmt.Errorf("detected different scales for BW send data: %s vs. %s (rank=%d, value=%f)", d.sBWUnit, scaledSendRankBWUnit, rank, d.sendRankBW[rank])
			}
			if d.sBWUnit == "" {
				d.sBWUnit = scaledSendRankBWUnit
			}
		}

		scaledRecvRankBW := d.recvRankBW
		if d.recvRankBW[rank] != 0 && !math.IsNaN(d.recvRankBW[rank]) {
			scaledRecvRankBWUnit := d.rBWUnit
			scaledRecvRankBWUnit, scaledRecvRankBW, err = scale.MapFloat64s("B/s", d.recvRankBW)
			if err != nil {
				return err
			}
			if d.rBWUnit != "" && d.rBWUnit != scaledRecvRankBWUnit {
				return fmt.Errorf("detected different scales for BW recv data: %s vs. %s (rank=%d, value=%f)", d.rBWUnit, scaledRecvRankBWUnit, rank, d.recvRankBW[rank])
			}
			if d.rBWUnit == "" {
				d.rBWUnit = scaledRecvRankBWUnit
			}
		}

		_, d.values = getMax(d.maxValue, d.values, rank, d.sendScaledHeatMap, d.recvScaledHeatMap, d.execScaledTimeMap, d.lateArrivalScaledTimeMap, scaledSendRankBW[rank], scaledRecvRankBW[rank])
		_, err = fd.WriteString(fmt.Sprintf("%d %d %d %f %f %f %f\n", rank, d.sendScaledHeatMap[rank], d.recvScaledHeatMap[rank], d.execScaledTimeMap[rank], d.lateArrivalScaledTimeMap[rank], scaledSendRankBW[rank], scaledRecvRankBW[rank]))
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *plotData) generateHostAvgs(hostname string) error {
	hostFile := filepath.Join(d.outputDir, hostname+"_avgs.txt")

	fd, err := os.OpenFile(hostFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.WriteString("# Rank send_size recv_size exec_time late_time send_bw recv_bw\n")
	if err != nil {
		return err
	}

	ranks := d.hostMap[hostname]
	d.numRanks = len(ranks)
	sort.Ints(ranks)
	for i := 0; i < d.emptyLines; i++ {
		_, err = fd.WriteString("- - - - - - -\n")
		if err != nil {
			return err
		}
	}
	for _, rank := range ranks {
		d.sendRankBW[rank] = float64(d.avgSendHeatMap[rank]) / d.avgExecTimeMap[rank]
		d.recvRankBW[rank] = float64(d.avgRecvHeatMap[rank]) / d.avgExecTimeMap[rank]

		var scaledSendRankBWUnit string
		var scaledRecvRankBWUnit string
		scaledSendRankBWUnit, scaledSendBW, err := scale.Float64s("B/s", []float64{d.sendRankBW[rank]})
		if err != nil {
			return err
		}
		d.scaledSendRankBW[rank] = scaledSendBW[0]
		scaledRecvRankBWUnit, scaledRecvBW, err := scale.Float64s("B/s", []float64{d.recvRankBW[rank]})
		if err != nil {
			return err
		}
		d.scaledRecvRankBW[rank] = scaledRecvBW[0]

		// If the average is different from 0, we try to scale it and hope that the scale
		// will match what we already have for other values. If not, we fail, we have no
		// mechanism to put various data to the same scale at the moment.
		// So, before starting to do some calculation, we assume the following default values
		// which are used when the average is equal to 0:
		// - the scaled BW is equal to non-scaled BW
		// - the unit is the one previous detected (by default the average is assumed to be
		// 	 equal to 0 so it does not matter)
		// Also note that if the rank is not in the communicator, the value is set to 'NaN'

		if d.sendRankBW[rank] != 0 && !math.IsNaN(d.sendRankBW[rank]) {
			if d.sBWUnit != "" && d.sBWUnit != scaledSendRankBWUnit {
				return fmt.Errorf("detected different scales for BW data")
			}
			if d.sBWUnit == "" {
				d.sBWUnit = scaledSendRankBWUnit
			}
		}

		if d.recvRankBW[rank] != 0 && !math.IsNaN(d.recvRankBW[rank]) {
			if d.rBWUnit != "" && d.rBWUnit != scaledRecvRankBWUnit {
				return fmt.Errorf("detected different scales for BW data")
			}
			if d.rBWUnit == "" {
				d.rBWUnit = scaledRecvRankBWUnit
			}
		}

		_, d.values = getMax(d.maxValue, d.values, rank, d.avgSendScaledHeatMap, d.avgRecvScaledHeatMap, d.avgExecScaledTimeMap, d.avgLateArrivalScaledTimeMap, d.sendRankBW[rank], d.recvRankBW[rank])
		_, err = fd.WriteString(fmt.Sprintf("%d %d %d %f %f %f %f\n", rank, d.avgSendScaledHeatMap[rank], d.avgRecvScaledHeatMap[rank], d.avgExecScaledTimeMap[rank], d.avgLateArrivalScaledTimeMap[rank], d.sendRankBW[0], d.recvRankBW[1]))
		if err != nil {
			return err
		}
	}
	return nil
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
	data := plotData{
		outputDir:             outputDir,
		hostMap:               hostMap,
		avgSendHeatMap:        avgSendHeatMap,
		avgRecvHeatMap:        avgRecvHeatMap,
		avgExecTimeMap:        avgExecTimeMap,
		avgLateArrivalTimeMap: avgLateArrivalTimeMap,
		maxValue:              1000, // We automatically scale the data, the max is always 1000
		values:                make(map[int]bool),
		sendRankBW:            make(map[int]float64),
		recvRankBW:            make(map[int]float64),
		scaledSendRankBW:      make(map[int]float64),
		scaledRecvRankBW:      make(map[int]float64),
		emptyLines:            0,
	}

	var err error
	data.avgSendHeatMapUnit, data.avgSendScaledHeatMap, err = scale.MapInts("B", avgSendHeatMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapInts() on avgSendHeatMap failed(): %s", err)
	}
	data.avgRecvHeatMapUnit, data.avgRecvScaledHeatMap, err = scale.MapInts("B", avgRecvHeatMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapInts() on avgRecvHeatMap failed(): %s", err)
	}
	data.avgExecTimeMapUnit, data.avgExecScaledTimeMap, err = scale.MapFloat64s("seconds", avgExecTimeMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapFloat64s() on avgExecTimeMap failed(): %s", err)
	}
	data.avgLateArrivalTimeMapUnit, data.avgLateArrivalScaledTimeMap, err = scale.MapFloat64s("seconds", avgLateArrivalTimeMap)
	if err != nil {
		return "", fmt.Errorf("scale.MapFloat64s() on avgLateArrivalTimeMap failed(): %s", err)
	}

	// fixme: atm we assume that all BW data is homogeneous so once we figure out a scale, it
	// is the same scale all the time. It might not be true so we really need to figure out the
	// scale based on sendHeatMapUnit and recvHeatMapUnit and force it to be used later when
	// calculating the bandwidth
	data.sBWUnit = ""
	data.rBWUnit = ""

	data.emptyLines = 0
	for _, hostname := range hosts {
		err = data.generateHostAvgs(hostname)
		if err != nil {
			return "", err
		}
		data.emptyLines += data.numRanks
	}

	data.emptyLines = 0
	idx := 0
	for _, hostname := range hosts {
		n, i, err := data.generateRanksMap(idx, hostname)
		if err != nil {
			return "", nil
		}
		idx = i
		data.emptyLines += n
	}

	gnuplotScript, err := generateGlobalPlotScript(outputDir, data.numRanks, data.maxValue, hosts, data.avgSendHeatMapUnit, data.avgRecvHeatMapUnit, data.avgExecTimeMapUnit, data.avgLateArrivalTimeMapUnit, data.sBWUnit, data.rBWUnit)
	if err != nil {
		return "", err
	}

	return gnuplotScript, nil
}

func getPlotFilename(leadRank int, callID int) string {
	return fmt.Sprintf("profiler_rank%d_call%d.png", leadRank, callID)
}

func getPlotDataFilePath(outputDir string, leadRank int, callID int, hostname string) string {
	return filepath.Join(outputDir, fmt.Sprintf("data_rank%d_call%d_%s.txt", leadRank, callID, hostname))
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
	data := plotData{
		outputDir:          outputDir,
		hostMap:            hostMap,
		sendHeatMap:        sendHeatMap,
		recvHeatMap:        recvHeatMap,
		execTimeMap:        execTimeMap,
		lateArrivalTimeMap: lateArrivalMap,
		maxValue:           1000, // We automatically scale the data, the max is always 1000
		values:             make(map[int]bool),
		sendRankBW:         make(map[int]float64),
		recvRankBW:         make(map[int]float64),
		scaledSendRankBW:   make(map[int]float64),
		scaledRecvRankBW:   make(map[int]float64),
		emptyLines:         0,
	}

	var err error
	data.sendHeatMapUnit, data.sendScaledHeatMap, err = scale.MapInts("B", sendHeatMap)
	if err != nil {
		return "", "", err
	}
	data.recvHeatMapUnit, data.recvScaledHeatMap, err = scale.MapInts("B", recvHeatMap)
	if err != nil {
		return "", "", err
	}
	data.execTimeMapUnit, data.execScaledTimeMap, err = scale.MapFloat64s("seconds", execTimeMap)
	if err != nil {
		return "", "", err
	}
	data.lateArrivalTimeMapUnit, data.lateArrivalScaledTimeMap, err = scale.MapFloat64s("seconds", lateArrivalMap)
	if err != nil {
		return "", "", err
	}

	data.sBWUnit = ""
	data.rBWUnit = ""

	data.emptyLines = 0
	for _, hostname := range hosts {
		err := data.generateCallsAvgs(hostname, leadRank, callID)
		if err != nil {
			return "", "", err
		}
		data.emptyLines += data.numRanks
	}

	data.emptyLines = 0
	idx := 0
	for _, hostname := range hosts {
		idx, err = data.generateHostCallData(hostname, idx)
		if err != nil {
			return "", "", err
		}
		data.emptyLines += len(hostMap[hostname])
	}

	var a []int
	for key := range data.values {
		a = append(a, key)
	}
	sort.Ints(a)

	pngFile, gnuplotScript, err := generateCallPlotScript(outputDir, leadRank, callID, data.numRanks, data.maxValue, a, hosts, data.sendHeatMapUnit, data.recvHeatMapUnit, data.execTimeMapUnit, data.lateArrivalTimeMapUnit, data.sBWUnit, data.rBWUnit)
	if err != nil {
		return "", "", fmt.Errorf("plot.generateCallPlotScript() failed: %s", err)
	}

	return pngFile, gnuplotScript, nil
}

func write(fd *os.File, dataFiles []string, numRanks int, maxValue int, hosts []string, sendUnit string, recvUnit string, execTimeUnit string, lateArrivalTimeUnit string, sendBWUnit string, recvBWUnit string) error {
	if len(hosts) == 0 {
		return fmt.Errorf("empty list of hosts")
	}
	_, err := fd.WriteString(fmt.Sprintf("set xrange [-1:%d]\n", numRanks))
	if err != nil {
		return err
	}
	_, err = fd.WriteString("set yrange [0:1000]\n")
	if err != nil {
		return err
	}
	_, err = fd.WriteString("set xtics\n\nset style fill pattern\n\nset style fill solid .1 noborder\nset style line 1 lc rgb 'black' pt 2\nset style line 2 lc rgb 'blue' pt 1\nset style line 3 lc rgb 'red' pt 4\nset style line 4 lc rgb 'pink' pt 9\nset style line 5 lc rgb 'green' pt 6\n")
	if err != nil {
		return err
	}
	_, err = fd.WriteString("\nshow label\n\n")
	if err != nil {
		return err
	}
	str := "plot "
	for _, hostname := range hosts {
		str += "\"ranks_map_" + hostname + ".txt\" using 0:1 with boxes title '" + hostname + "', \\\n"
	}

	// Special for the first node
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 2:xtic(1) with points ls 1 title \"data sent (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, sendUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 3 with points ls 2 title \"data received (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, recvUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 4 with points ls 3 title \"execution time (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, execTimeUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 5 with points ls 4 title \"late arrival timing (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, lateArrivalTimeUnit))
	str += fmt.Sprintf(fmt.Sprintf("\"%s.txt\" using 6 with points ls 5 title \"bandwidth (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, sendBWUnit))
	for i := 1; i < len(hosts); i++ {
		str += fmt.Sprintf("\"%s.txt\" using 2:xtic(1) with points ls 1 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s.txt\" using 3 with points ls 2 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s.txt\" using 4 with points ls 3 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s.txt\" using 5 with points ls 4 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s.txt\" using 6 with points ls 5 notitle, \\\n", dataFiles[i])
	}
	str = strings.TrimRight(str, ", \\\n")
	_, err = fd.WriteString(str)
	if err != nil {
		return err
	}

	return nil
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

	var dataFiles []string
	for _, hostname := range hosts {
		dataFiles = append(dataFiles, getPlotDataFilePath(outputDir, leadRank, callID, hostname))
	}
	err = write(fd, dataFiles, numRanks, maxValue, hosts, sendUnit, recvUnit, execTimeUnit, lateTimeUnit, sendBWUnit, recvBWUnit)
	if err != nil {
		return "", "", err
	}

	return targetPlotFile, plotScriptFile, nil
}

func generateGlobalPlotScript(outputDir string, numRanks int, maxValue int, hosts []string, sendUnit string, recvUnit string, execTimeUnit string, lateTimeUnit string, sendBWUnit string, recvBWUnit string) (string, error) {
	var dataFiles []string
	for _, hostname := range hosts {
		dataFiles = append(dataFiles, filepath.Join(outputDir, hostname+"_avgs.txt"))
	}
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
	err = write(fd, dataFiles, numRanks, maxValue, hosts, sendUnit, recvUnit, execTimeUnit, lateTimeUnit, sendBWUnit, recvBWUnit)
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

// Write into a file the "body" of the gnuplot
func writeHeatmaps(fd *os.File, HeatMatrix [][]int, outputDir string, numRanks int) error {

	// Matrix to plot
	str := "$map2 << EOD\n"
	for i := numRanks-1; i >= 0; i-- {
		for j := 0; j < numRanks; j++ {
			str += fmt.Sprintf(" %d", HeatMatrix[i][j])
		}
		str += "\n"
	}

	str += "EOD\n\n"

	// Extra information to display the plot correctly
	str += "set rmargin 10\n"
	str += "set yrange [-0.5:*]\n"
	str += "set xrange [-0.5:*]\n"
	str += "set cbrange [0:7]\n"
	str += "set xlabel \"Senders\"\n"
	str += "set ylabel \"Receivers\"\n"
	if numRanks < 40 {
		str += "set xtics 1\n"
		str += "set ytics 1\n"
	} else {
		str += "set xtics rotate by 22.5\n"
		str += "set xtics 5 offset 0,-0.75,0\n"
		str += "set ytics 5\n"
	}
	str += "set cbtics (\"0\" 0,\"1-10\" 1,\"11-100\" 2,\"101-1,000\" 3,\"1,001-10,000\" 4,\"10,001-100,000\" 5,\"100,001-1,000,000\" 6,\"1,000,001-max\" 7)\n"
	str += "set palette defined (0 \"white\", 0.5 \"white\", 0.5 \"yellow\", 1.5 \"yellow\", 1.5 \"orange\", 2.5 \"orange\", 2.5 \"green\", 3.5 \"green\", 3.5 \"red\", 4.5 \"red\", 4.5 \"purple\", 5.5 \"purple\", 5.5 \"brown\", 6.5 \"brown\", 6.5 \"black\", 7 \"black\")\n"
	str += "unset key\n"
	str += "plot '$map2' using 1:2:3 matrix with image\n"

	_, err := fd.WriteString(str)
	if err != nil {
		return err
	}

	return nil
}

// Creates the gnuplot script of a pattern
func generateTask3Plots(outputDir string, HeatMatrix [][]int, numRanks, numPattern int, NumberOfCalls string) (string, error) {
	
	namePlot := fmt.Sprintf("heatmap-task3-pattern%d.gnuplot", numPattern)
	plotScriptFile := filepath.Join(outputDir, namePlot)
	fd, err := os.OpenFile(plotScriptFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	// Gnuplot "header"
	str := "set term png size 1200,900\n"
	str += fmt.Sprintf("set output \"heatmap-task3-pattern%d.png\"\n\n", numPattern)
	str += fmt.Sprintf("set title \"Heatmap of Pattern %d: %s calls.\"\n\n", numPattern, NumberOfCalls)

	_, err = fd.WriteString(str)
	if err != nil {
		return "", err
	}
	// Gnuplot "body"
	err = writeHeatmaps(fd, HeatMatrix, outputDir, numRanks)
	if err != nil {
		return "", err
	}

	return plotScriptFile, nil
}

// Generation of heatmaps plots of Task 3
func Task3(outputDir string, SendDataForTask3 []map[int][]int, NumberOfCalls []string) error {
	
	// Total of patterns
	var numPatterns = len(SendDataForTask3)

	// If there are no patterns, nothing is displayed
	if numPatterns == 0 {
		return nil
	}

	// Total of ranks of the pattern calls
	var numRanks = len(SendDataForTask3[0])

	// For every pattern, we map the info of the original matrix to a matrix to print with gnuplot
	for i := 0; i < numPatterns; i++ {
		// Allocate and initialize matrix to put inside the plot
		HeatMatrix := make([][]int, numRanks)
		for ii := 0; ii < numRanks; ii++ {
		    HeatMatrix[ii] = make([]int, numRanks)
		}
		// Prepare matrix with mapped values
		for j := 0; j < numRanks; j++ {
			for k := 0; k < numRanks; k++ {
				OriginalValue := SendDataForTask3[numPatterns-1-i][j][k]
				NewValue := 0
				switch {
			    case OriginalValue == 0:
			        NewValue = 0
			    case 0 < OriginalValue && OriginalValue <= 10:
			        NewValue = 1
			    case 10 < OriginalValue && OriginalValue <= 100:
			        NewValue = 2
			    case 100 < OriginalValue && OriginalValue <= 1000:
			        NewValue = 3
			    case 1000 < OriginalValue && OriginalValue <= 10000:
			        NewValue = 4
			    case 10000 < OriginalValue && OriginalValue <= 100000:
			        NewValue = 5
			    case 100000 < OriginalValue && OriginalValue <= 1000000:
			        NewValue = 6
			    case 1000000 < OriginalValue:
			        NewValue = 7
			    }

			    // We start putting values from the bottom row to generate the plot correctly
			    HeatMatrix[numRanks-1-k][j] = NewValue
			}
		}
		// Generate the plot script with all the previous info
		gnuplotScript, err := generateTask3Plots(outputDir, HeatMatrix, numRanks, i, NumberOfCalls[numPatterns-1-i])
		if err != nil {
			return fmt.Errorf("generateTask3Plots() failed: %s", err)
		}
		runGnuplot(gnuplotScript, outputDir)
	}

	return nil
}

// Creates the gnuplot script of a pattern for task4
func generateTask4Plots(outputDir string, HeatMatrix [][]int, numRanks int) (string, error) {
	
	namePlot := fmt.Sprintf("heatmap-task4-allpatterns.gnuplot")
	plotScriptFile := filepath.Join(outputDir, namePlot)
	fd, err := os.OpenFile(plotScriptFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	// Gnuplot "header"
	str := "set term png size 1200,900\n"
	str += fmt.Sprintf("set output \"heatmap-task4-allpatterns.png\"\n\n")
	str += fmt.Sprintf("set title \"Heatmap of the sum of all patterns.\"\n\n")

	_, err = fd.WriteString(str)
	if err != nil {
		return "", err
	}
	// Gnuplot "body"
	err = writeHeatmaps(fd, HeatMatrix, outputDir, numRanks)
	if err != nil {
		return "", err
	}

	return plotScriptFile, nil
}

// Generation of the heatmap plot of Task 4
func Task4(outputDir string, SendDataForTask3 []map[int][]int, NumberOfCalls []string) error {
	
	// Total of patterns
	var numPatterns = len(SendDataForTask3)

	// If there are no patterns, nothing is displayed
	if numPatterns == 0 {
		return nil
	}

	// Total of ranks of the pattern calls
	var numRanks = len(SendDataForTask3[0])

	// Allocate and initialize matrix to put inside the plot
	HeatMatrix    := make([][]int,     numRanks)
	HeatMatrixAux := make([][]float64, numRanks)
	for ii := 0; ii < numRanks; ii++ {
	    HeatMatrix[ii]    = make([]int,     numRanks)
	    HeatMatrixAux[ii] = make([]float64, numRanks)
	}

	// Calcule the weighted sum of the patterns
	for i := 0; i < numPatterns; i++ {
		weightSplitted := strings.Split(NumberOfCalls[numPatterns-1-i], "/")
		weight, _ := strconv.Atoi(weightSplitted[0])
		totalWeight, _ := strconv.Atoi(weightSplitted[1])
		// Prepare matrix with mapped values
		for j := 0; j < numRanks; j++ {
			for k := 0; k < numRanks; k++ {
				OriginalValue := SendDataForTask3[numPatterns-1-i][j][k]
			    // We start putting values from the bottom row to generate the plot correctly
			    HeatMatrixAux[numRanks-1-k][j] += float64(OriginalValue)*(float64(weight)/float64(totalWeight))
			}
		}
	}

	// Prepare matrix with mapped values
	for j := 0; j < numRanks; j++ {
		for k := 0; k < numRanks; k++ {
			OriginalValue := HeatMatrixAux[j][k]
			NewValue := 0
			switch {
		    case OriginalValue == 0:
		        NewValue = 0
		    case 0 < OriginalValue && OriginalValue <= 10:
		        NewValue = 1
		    case 10 < OriginalValue && OriginalValue <= 100:
		        NewValue = 2
		    case 100 < OriginalValue && OriginalValue <= 1000:
		        NewValue = 3
		    case 1000 < OriginalValue && OriginalValue <= 10000:
		        NewValue = 4
		    case 10000 < OriginalValue && OriginalValue <= 100000:
		        NewValue = 5
		    case 100000 < OriginalValue && OriginalValue <= 1000000:
		        NewValue = 6
		    case 1000000 < OriginalValue:
		        NewValue = 7
		    }
		    HeatMatrix[k][j] = NewValue
		}
	}

	// Generate the plot script with all the previous info
	gnuplotScript, err := generateTask4Plots(outputDir, HeatMatrix, numRanks)
	if err != nil {
		return fmt.Errorf("generateTask3Plots() failed: %s", err)
	}
	runGnuplot(gnuplotScript, outputDir)

	return nil
}
