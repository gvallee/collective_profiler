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

	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/location"
	"github.com/gvallee/alltoallv_profiling/tools/internal/pkg/patterns"
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
		if _, ok := d.execTimeMap[rank]; !ok {
			// exec time not found, avoid division with zero
			continue
		}

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
		if _, ok := d.avgExecTimeMap[rank]; !ok {
			// exec time not found, avoid division with zero
			continue
		}

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
	str += fmt.Sprintf("\"%s\" using 2:xtic(1) with points ls 1 title \"data sent (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, sendUnit)
	str += fmt.Sprintf("\"%s\" using 3 with points ls 2 title \"data received (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, recvUnit)
	str += fmt.Sprintf("\"%s\" using 4 with points ls 3 title \"execution time (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, execTimeUnit)
	str += fmt.Sprintf("\"%s\" using 5 with points ls 4 title \"late arrival timing (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, lateArrivalTimeUnit)
	str += fmt.Sprintf("\"%s\" using 6 with points ls 5 title \"bandwidth (%s)\", \\\n", dataFiles[0] /*filepath.Base(getPlotDataFilePath(outputDir, leadRank, callID, hosts[0]))*/, sendBWUnit)
	for i := 1; i < len(hosts); i++ {
		str += fmt.Sprintf("\"%s\" using 2:xtic(1) with points ls 1 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s\" using 3 with points ls 2 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s\" using 4 with points ls 3 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s\" using 5 with points ls 4 notitle, \\\n", dataFiles[i])
		str += fmt.Sprintf("\"%s\" using 6 with points ls 5 notitle, \\\n", dataFiles[i])
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

type heavyPatternWithLeadRank struct {
	leadRank int
	pattern  patterns.HeavyPattern
}

func generateHeavyPatternsDataFiles(dir string, outputDir string, allPatterns map[int]patterns.Data) ([]string, error) {
	// collect patterns from different communicators
	heavyPatterns := make([]heavyPatternWithLeadRank, 0)
	for leadRank, data := range allPatterns {
		for _, pattern := range data.HeavyPatterns {
			heavyPatterns = append(heavyPatterns, heavyPatternWithLeadRank{
				leadRank: leadRank,
				pattern:  pattern,
			})
		}
	}

	// sort by occurrence
	sort.Slice(heavyPatterns, func(i, j int) bool {
		return heavyPatterns[i].pattern.Occurrence > heavyPatterns[j].pattern.Occurrence
	})

	// 10 most heavy patterns
	if len(heavyPatterns) > 10 {
		heavyPatterns = heavyPatterns[:10]
	}

	gnuplotFiles := make([]string, 0)

	for _, dist := range AllDists {
		for i, heavyPattern := range heavyPatterns {
			// find min/max value
			maxBytes := 0
			minBytes := math.MaxInt32
			for _, ranks := range heavyPattern.pattern.Counts {
				for _, value := range ranks {
					if maxBytes < value {
						maxBytes = value
					}
					if minBytes > value {
						minBytes = value
					}
				}
			}

			// dump heat map data
			dataFile := filepath.Join(outputDir, fmt.Sprintf("heavy_patterns_index%d_%s.txt", i, dist.Name()))
			fd, err := os.OpenFile(dataFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return nil, err
			}
			defer fd.Close()

			ranks := make([]int, 0)
			for rank := range heavyPattern.pattern.Counts {
				ranks = append(ranks, rank)
			}
			sort.Ints(ranks)

			// xlabels
			var labels strings.Builder
			lastRank := 0
			for _, rank := range ranks {
				// skip consecutive ranks
				if rank != lastRank+1 || rank == ranks[len(ranks)-1] {
					fmt.Fprintf(&labels, ",%d", rank)
				} else {
					fmt.Fprintf(&labels, ",")
				}
				lastRank = rank
			}
			_, err = fd.WriteString(fmt.Sprintf("%s\n", labels.String()))
			if err != nil {
				return nil, err
			}

			// heat map matrix
			lastRank = 0
			for _, rank := range ranks {
				var row strings.Builder
				// skip consecutive ranks
				if rank != lastRank+1 || rank == ranks[len(ranks)-1] {
					fmt.Fprintf(&row, "%d", rank)
				}
				lastRank = rank

				for _, value := range heavyPattern.pattern.Counts[rank] {
					// convert value range to color index
					color := dist.Map(value, maxBytes)

					fmt.Fprintf(&row, ",%d", color)
				}

				_, err = fd.WriteString(fmt.Sprintf("%s\n", row.String()))
				if err != nil {
					return nil, err
				}
			}

			// dump gnuplot script
			gnuplotFile := filepath.Join(outputDir, fmt.Sprintf("heavy_patterns_index%d_%s.gnuplot", i, dist.Name()))
			fd, err = os.OpenFile(gnuplotFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return nil, err
			}
			defer fd.Close()

			_, err = fd.WriteString(fmt.Sprintf(`set term png
set output "heavy_patterns_index%d_%s.png"
set title "Heat map Comm %d Top %d Occurrences %d"
set size ratio 1
set xlabel "Send Rank"
set ylabel "Recv Rank"
unset key
set xrange [-0.5:%d.5]
set yrange [-0.5:%d.5]
set pointsize 2
set datafile separator comma
%s
plot "heavy_patterns_index%d_%s.txt" matrix rowheaders columnheaders using 2:1:3 with image `, i, dist.Name(), heavyPattern.leadRank, i+1, heavyPattern.pattern.Occurrence, len(ranks)-1, len(ranks)-1, dist.GnuplotConfig(minBytes, maxBytes), i, dist.Name()))
			if err != nil {
				return nil, err
			}

			gnuplotFiles = append(gnuplotFiles, gnuplotFile)
		}
	}
	return gnuplotFiles, nil
}

// HeavyPatterns plots the heavy patterns found during the post-mortem analysis
func HeavyPatterns(dir string, outputDir string, patterns map[int]patterns.Data) error {
	gnuplotScripts, err := generateHeavyPatternsDataFiles(dir, outputDir, patterns)
	if err != nil {
		return fmt.Errorf("generateHeavyPatternsDataFiles() failed: %s", err)
	}

	for _, gnuplotScript := range gnuplotScripts {
		err = runGnuplot(gnuplotScript, outputDir)
		if err != nil {
			return fmt.Errorf("runGnuplot() failed: %s", err)
		}
	}

	return nil
}

type Distribution interface {
	Map(bytes int, maxBytes int) int
	GnuplotConfig(minBytes int, maxBytes int) string
	Name() string
}

var AllDists = []Distribution{
	SimpleDistribution{},
	LinearDistribution{},
	LogarithmDistribution{},
	Linear2Distribution{},
	LinearViridisDistribution{},
	QuadraticDistribution{},
}

func generateAllPatternsDataFiles(dir string, outputDir string, numRanks int, allPatterns map[int]patterns.Data, locationsData []*location.Data) ([]string, error) {
	// create numRanks x numRanks matrix
	matrix := make([][]int, numRanks)
	for i := 0; i < numRanks; i++ {
		matrix[i] = make([]int, numRanks)
	}

	// create mapping from local rank to COMM_WORLD ranks
	mapping := make(map[int]map[int]int)
	for _, data := range locationsData {
		leadRank := data.RankLocations[0].CommWorldRank
		mapping[leadRank] = make(map[int]int)
		for _, loc := range data.RankLocations {
			mapping[leadRank][loc.CommRank] = loc.CommWorldRank
		}
	}

	// sum up patterns from different communicators
	for leadRank, data := range allPatterns {
		for _, pattern := range data.HeavyPatterns {
			for from, value := range pattern.Counts {
				for to, bytes := range value {
					// convert `from` and `to` to COMM_WORLD ranks
					world_from := mapping[leadRank][from]
					world_to := mapping[leadRank][to]
					matrix[world_from][world_to] += bytes * pattern.Occurrence
				}
			}
		}
	}

	// find min/max value
	maxBytes := matrix[0][0]
	minBytes := matrix[0][0]
	for rank := 0; rank < numRanks; rank++ {
		for to := 0; to < numRanks; to++ {
			if maxBytes < matrix[rank][to] {
				maxBytes = matrix[rank][to]
			}
			if minBytes > matrix[rank][to] {
				minBytes = matrix[rank][to]
			}
		}
	}

	gnuplotFiles := make([]string, 0)

	for _, dist := range AllDists {
		// dump heat map data
		dataFile := filepath.Join(outputDir, fmt.Sprintf("all_patterns_%s.txt", dist.Name()))
		fd, err := os.OpenFile(dataFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		// xlabels
		var labels strings.Builder
		for rank := 0; rank < numRanks; rank++ {
			if rank == 0 || rank == numRanks-1 {
				fmt.Fprintf(&labels, ",%d", rank)
			} else {
				fmt.Fprintf(&labels, ",")
			}
		}
		_, err = fd.WriteString(fmt.Sprintf("%s\n", labels.String()))
		if err != nil {
			return nil, err
		}

		// heat map matrix
		for rank := 0; rank < numRanks; rank++ {
			var row strings.Builder
			if rank == 0 || rank == numRanks-1 {
				fmt.Fprintf(&row, "%d", rank)
			}

			for to := 0; to < numRanks; to++ {
				// convert value range to color index
				value := matrix[rank][to]
				color := dist.Map(value, maxBytes)

				fmt.Fprintf(&row, ",%d", color)
			}

			_, err = fd.WriteString(fmt.Sprintf("%s\n", row.String()))
			if err != nil {
				return nil, err
			}
		}

		// dump gnuplot script
		gnuplotFile := filepath.Join(outputDir, fmt.Sprintf("all_patterns_%s.gnuplot", dist.Name()))
		fd, err = os.OpenFile(gnuplotFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		_, err = fd.WriteString(fmt.Sprintf(`set term png
set output "all_patterns_%s.png"
set title "Heat map of sum of all patterns"
set size ratio 1
set xlabel "Send Rank"
set ylabel "Recv Rank"
unset key
set xrange [-0.5:%d.5]
set yrange [-0.5:%d.5]
set pointsize 2
set datafile separator comma
%s
plot "all_patterns_%s.txt" matrix rowheaders columnheaders using 2:1:3 with image `, dist.Name(), numRanks-1, numRanks-1, dist.GnuplotConfig(minBytes, maxBytes), dist.Name()))
		if err != nil {
			return nil, err
		}

		gnuplotFiles = append(gnuplotFiles, gnuplotFile)
	}
	return gnuplotFiles, nil
}

// distributions
type SimpleDistribution struct{}

func (d SimpleDistribution) Map(value int, maxBytes int) int {
	// 0 'white', 1 'yellow', 2 'orange', 3 'green', 4 'red', 5 'purple', 6 'brown', 7 'black'
	color := 0
	if value > 0 {
		value = (value - 1) / 10
		color += 1
		for value > 0 {
			value /= 10
			color += 1
		}
	}
	return color
}

func (d SimpleDistribution) GnuplotConfig(minBytes int, maxBytes int) string {
	// filled contour not working
	// offset by 0.0001
	// https://stackoverflow.com/questions/33955878/managing-the-palette-indicators-in-gnuplot
	return `set palette defined (0 'white', 0.0001 'white', 0.0002 'yellow', 0.9999 'yellow', 1.0001 'orange', 1.9999 'orange', 2.0001 'green', 2.9999 'green', 3.0001 'red', 3.9999 'red', 4.0001 'purple', 4.9999 'purple', 5.0001 'brown', 5.9999 'brown', 6.0001 'black', 7 'black')
		set cbrange [0:7]
		set palette maxcolors 8
		set cbtics ("0" 0, "10" 1, "100" 2, "1000" 3, "10000" 4, "100000" 5, "1000000" 6, "infinity" 7)`
}

func (d SimpleDistribution) Name() string {
	return "simple"
}

type LinearDistribution struct{}

func (d LinearDistribution) Map(value int, maxBytes int) int {
	return value
}

func (d LinearDistribution) GnuplotConfig(minBytes int, maxBytes int) string {
	b := strings.Builder{}
	fmt.Fprintf(&b, "set palette defined( 0 'white', %d 'black' )\n", maxBytes)
	fmt.Fprintf(&b, "set cbrange [0:%d]\n", maxBytes)
	return b.String()
}

func (d LinearDistribution) Name() string {
	return "linear"
}

type Linear2Distribution struct{}

func (d Linear2Distribution) Map(value int, maxBytes int) int {
	return value
}

func (d Linear2Distribution) GnuplotConfig(minBytes int, maxBytes int) string {
	b := strings.Builder{}
	fmt.Fprintf(&b, "set palette defined( %d 'white', %d 'black' )\n", minBytes, maxBytes)
	fmt.Fprintf(&b, "set cbrange [%d:%d]\n", minBytes, maxBytes)
	return b.String()
}

func (d Linear2Distribution) Name() string {
	return "linear2"
}

// https://github.com/Gnuplotting/gnuplot-palettes/blob/master/viridis.pal
type LinearViridisDistribution struct{}

func (d LinearViridisDistribution) Map(value int, maxBytes int) int {
	return value * 256 / maxBytes
}

func (d LinearViridisDistribution) GnuplotConfig(minBytes int, maxBytes int) string {
	b := strings.Builder{}
	fmt.Fprintf(&b, `# New matplotlib colormaps by Nathaniel J. Smith, Stefan van der Walt,
	# and (in the case of viridis) Eric Firing.
	#
	# This file and the colormaps in it are released under the CC0 license /
	# public domain dedication. We would appreciate credit if you use or
	# redistribute these colormaps, but do not impose any legal restrictions.
	#
	# To the extent possible under law, the persons who associated CC0 with
	# mpl-colormaps have waived all copyright and related or neighboring rights
	# to mpl-colormaps.
	#
	# You should have received a copy of the CC0 legalcode along with this
	# work.  If not, see <http://creativecommons.org/publicdomain/zero/1.0/>.
	
	#https://github.com/BIDS/colormap/blob/master/colormaps.py
	
	
	# line styles
	set style line  1 lt 1 lc rgb '#440154' # dark purple
	set style line  2 lt 1 lc rgb '#472c7a' # purple
	set style line  3 lt 1 lc rgb '#3b518b' # blue
	set style line  4 lt 1 lc rgb '#2c718e' # blue
	set style line  5 lt 1 lc rgb '#21908d' # blue-green
	set style line  6 lt 1 lc rgb '#27ad81' # green
	set style line  7 lt 1 lc rgb '#5cc863' # green
	set style line  8 lt 1 lc rgb '#aadc32' # lime green
	set style line  9 lt 1 lc rgb '#fde725' # yellow
	
	
	# palette
	set palette defined (\
	0   0.267004 0.004874 0.329415,\
	1   0.268510 0.009605 0.335427,\
	2   0.269944 0.014625 0.341379,\
	3   0.271305 0.019942 0.347269,\
	4   0.272594 0.025563 0.353093,\
	5   0.273809 0.031497 0.358853,\
	6   0.274952 0.037752 0.364543,\
	7   0.276022 0.044167 0.370164,\
	8   0.277018 0.050344 0.375715,\
	9   0.277941 0.056324 0.381191,\
	10  0.278791 0.062145 0.386592,\
	11  0.279566 0.067836 0.391917,\
	12  0.280267 0.073417 0.397163,\
	13  0.280894 0.078907 0.402329,\
	14  0.281446 0.084320 0.407414,\
	15  0.281924 0.089666 0.412415,\
	16  0.282327 0.094955 0.417331,\
	17  0.282656 0.100196 0.422160,\
	18  0.282910 0.105393 0.426902,\
	19  0.283091 0.110553 0.431554,\
	20  0.283197 0.115680 0.436115,\
	21  0.283229 0.120777 0.440584,\
	22  0.283187 0.125848 0.444960,\
	23  0.283072 0.130895 0.449241,\
	24  0.282884 0.135920 0.453427,\
	25  0.282623 0.140926 0.457517,\
	26  0.282290 0.145912 0.461510,\
	27  0.281887 0.150881 0.465405,\
	28  0.281412 0.155834 0.469201,\
	29  0.280868 0.160771 0.472899,\
	30  0.280255 0.165693 0.476498,\
	31  0.279574 0.170599 0.479997,\
	32  0.278826 0.175490 0.483397,\
	33  0.278012 0.180367 0.486697,\
	34  0.277134 0.185228 0.489898,\
	35  0.276194 0.190074 0.493001,\
	36  0.275191 0.194905 0.496005,\
	37  0.274128 0.199721 0.498911,\
	38  0.273006 0.204520 0.501721,\
	39  0.271828 0.209303 0.504434,\
	40  0.270595 0.214069 0.507052,\
	41  0.269308 0.218818 0.509577,\
	42  0.267968 0.223549 0.512008,\
	43  0.266580 0.228262 0.514349,\
	44  0.265145 0.232956 0.516599,\
	45  0.263663 0.237631 0.518762,\
	46  0.262138 0.242286 0.520837,\
	47  0.260571 0.246922 0.522828,\
	48  0.258965 0.251537 0.524736,\
	49  0.257322 0.256130 0.526563,\
	50  0.255645 0.260703 0.528312,\
	51  0.253935 0.265254 0.529983,\
	52  0.252194 0.269783 0.531579,\
	53  0.250425 0.274290 0.533103,\
	54  0.248629 0.278775 0.534556,\
	55  0.246811 0.283237 0.535941,\
	56  0.244972 0.287675 0.537260,\
	57  0.243113 0.292092 0.538516,\
	58  0.241237 0.296485 0.539709,\
	59  0.239346 0.300855 0.540844,\
	60  0.237441 0.305202 0.541921,\
	61  0.235526 0.309527 0.542944,\
	62  0.233603 0.313828 0.543914,\
	63  0.231674 0.318106 0.544834,\
	64  0.229739 0.322361 0.545706,\
	65  0.227802 0.326594 0.546532,\
	66  0.225863 0.330805 0.547314,\
	67  0.223925 0.334994 0.548053,\
	68  0.221989 0.339161 0.548752,\
	69  0.220057 0.343307 0.549413,\
	70  0.218130 0.347432 0.550038,\
	71  0.216210 0.351535 0.550627,\
	72  0.214298 0.355619 0.551184,\
	73  0.212395 0.359683 0.551710,\
	74  0.210503 0.363727 0.552206,\
	75  0.208623 0.367752 0.552675,\
	76  0.206756 0.371758 0.553117,\
	77  0.204903 0.375746 0.553533,\
	78  0.203063 0.379716 0.553925,\
	79  0.201239 0.383670 0.554294,\
	80  0.199430 0.387607 0.554642,\
	81  0.197636 0.391528 0.554969,\
	82  0.195860 0.395433 0.555276,\
	83  0.194100 0.399323 0.555565,\
	84  0.192357 0.403199 0.555836,\
	85  0.190631 0.407061 0.556089,\
	86  0.188923 0.410910 0.556326,\
	87  0.187231 0.414746 0.556547,\
	88  0.185556 0.418570 0.556753,\
	89  0.183898 0.422383 0.556944,\
	90  0.182256 0.426184 0.557120,\
	91  0.180629 0.429975 0.557282,\
	92  0.179019 0.433756 0.557430,\
	93  0.177423 0.437527 0.557565,\
	94  0.175841 0.441290 0.557685,\
	95  0.174274 0.445044 0.557792,\
	96  0.172719 0.448791 0.557885,\
	97  0.171176 0.452530 0.557965,\
	98  0.169646 0.456262 0.558030,\
	99  0.168126 0.459988 0.558082,\
	100 0.166617 0.463708 0.558119,\
	101 0.165117 0.467423 0.558141,\
	102 0.163625 0.471133 0.558148,\
	103 0.162142 0.474838 0.558140,\
	104 0.160665 0.478540 0.558115,\
	105 0.159194 0.482237 0.558073,\
	106 0.157729 0.485932 0.558013,\
	107 0.156270 0.489624 0.557936,\
	108 0.154815 0.493313 0.557840,\
	109 0.153364 0.497000 0.557724,\
	110 0.151918 0.500685 0.557587,\
	111 0.150476 0.504369 0.557430,\
	112 0.149039 0.508051 0.557250,\
	113 0.147607 0.511733 0.557049,\
	114 0.146180 0.515413 0.556823,\
	115 0.144759 0.519093 0.556572,\
	116 0.143343 0.522773 0.556295,\
	117 0.141935 0.526453 0.555991,\
	118 0.140536 0.530132 0.555659,\
	119 0.139147 0.533812 0.555298,\
	120 0.137770 0.537492 0.554906,\
	121 0.136408 0.541173 0.554483,\
	122 0.135066 0.544853 0.554029,\
	123 0.133743 0.548535 0.553541,\
	124 0.132444 0.552216 0.553018,\
	125 0.131172 0.555899 0.552459,\
	126 0.129933 0.559582 0.551864,\
	127 0.128729 0.563265 0.551229,\
	128 0.127568 0.566949 0.550556,\
	129 0.126453 0.570633 0.549841,\
	130 0.125394 0.574318 0.549086,\
	131 0.124395 0.578002 0.548287,\
	132 0.123463 0.581687 0.547445,\
	133 0.122606 0.585371 0.546557,\
	134 0.121831 0.589055 0.545623,\
	135 0.121148 0.592739 0.544641,\
	136 0.120565 0.596422 0.543611,\
	137 0.120092 0.600104 0.542530,\
	138 0.119738 0.603785 0.541400,\
	139 0.119512 0.607464 0.540218,\
	140 0.119423 0.611141 0.538982,\
	141 0.119483 0.614817 0.537692,\
	142 0.119699 0.618490 0.536347,\
	143 0.120081 0.622161 0.534946,\
	144 0.120638 0.625828 0.533488,\
	145 0.121380 0.629492 0.531973,\
	146 0.122312 0.633153 0.530398,\
	147 0.123444 0.636809 0.528763,\
	148 0.124780 0.640461 0.527068,\
	149 0.126326 0.644107 0.525311,\
	150 0.128087 0.647749 0.523491,\
	151 0.130067 0.651384 0.521608,\
	152 0.132268 0.655014 0.519661,\
	153 0.134692 0.658636 0.517649,\
	154 0.137339 0.662252 0.515571,\
	155 0.140210 0.665859 0.513427,\
	156 0.143303 0.669459 0.511215,\
	157 0.146616 0.673050 0.508936,\
	158 0.150148 0.676631 0.506589,\
	159 0.153894 0.680203 0.504172,\
	160 0.157851 0.683765 0.501686,\
	161 0.162016 0.687316 0.499129,\
	162 0.166383 0.690856 0.496502,\
	163 0.170948 0.694384 0.493803,\
	164 0.175707 0.697900 0.491033,\
	165 0.180653 0.701402 0.488189,\
	166 0.185783 0.704891 0.485273,\
	167 0.191090 0.708366 0.482284,\
	168 0.196571 0.711827 0.479221,\
	169 0.202219 0.715272 0.476084,\
	170 0.208030 0.718701 0.472873,\
	171 0.214000 0.722114 0.469588,\
	172 0.220124 0.725509 0.466226,\
	173 0.226397 0.728888 0.462789,\
	174 0.232815 0.732247 0.459277,\
	175 0.239374 0.735588 0.455688,\
	176 0.246070 0.738910 0.452024,\
	177 0.252899 0.742211 0.448284,\
	178 0.259857 0.745492 0.444467,\
	179 0.266941 0.748751 0.440573,\
	180 0.274149 0.751988 0.436601,\
	181 0.281477 0.755203 0.432552,\
	182 0.288921 0.758394 0.428426,\
	183 0.296479 0.761561 0.424223,\
	184 0.304148 0.764704 0.419943,\
	185 0.311925 0.767822 0.415586,\
	186 0.319809 0.770914 0.411152,\
	187 0.327796 0.773980 0.406640,\
	188 0.335885 0.777018 0.402049,\
	189 0.344074 0.780029 0.397381,\
	190 0.352360 0.783011 0.392636,\
	191 0.360741 0.785964 0.387814,\
	192 0.369214 0.788888 0.382914,\
	193 0.377779 0.791781 0.377939,\
	194 0.386433 0.794644 0.372886,\
	195 0.395174 0.797475 0.367757,\
	196 0.404001 0.800275 0.362552,\
	197 0.412913 0.803041 0.357269,\
	198 0.421908 0.805774 0.351910,\
	199 0.430983 0.808473 0.346476,\
	200 0.440137 0.811138 0.340967,\
	201 0.449368 0.813768 0.335384,\
	202 0.458674 0.816363 0.329727,\
	203 0.468053 0.818921 0.323998,\
	204 0.477504 0.821444 0.318195,\
	205 0.487026 0.823929 0.312321,\
	206 0.496615 0.826376 0.306377,\
	207 0.506271 0.828786 0.300362,\
	208 0.515992 0.831158 0.294279,\
	209 0.525776 0.833491 0.288127,\
	210 0.535621 0.835785 0.281908,\
	211 0.545524 0.838039 0.275626,\
	212 0.555484 0.840254 0.269281,\
	213 0.565498 0.842430 0.262877,\
	214 0.575563 0.844566 0.256415,\
	215 0.585678 0.846661 0.249897,\
	216 0.595839 0.848717 0.243329,\
	217 0.606045 0.850733 0.236712,\
	218 0.616293 0.852709 0.230052,\
	219 0.626579 0.854645 0.223353,\
	220 0.636902 0.856542 0.216620,\
	221 0.647257 0.858400 0.209861,\
	222 0.657642 0.860219 0.203082,\
	223 0.668054 0.861999 0.196293,\
	224 0.678489 0.863742 0.189503,\
	225 0.688944 0.865448 0.182725,\
	226 0.699415 0.867117 0.175971,\
	227 0.709898 0.868751 0.169257,\
	228 0.720391 0.870350 0.162603,\
	229 0.730889 0.871916 0.156029,\
	230 0.741388 0.873449 0.149561,\
	231 0.751884 0.874951 0.143228,\
	232 0.762373 0.876424 0.137064,\
	233 0.772852 0.877868 0.131109,\
	234 0.783315 0.879285 0.125405,\
	235 0.793760 0.880678 0.120005,\
	236 0.804182 0.882046 0.114965,\
	237 0.814576 0.883393 0.110347,\
	238 0.824940 0.884720 0.106217,\
	239 0.835270 0.886029 0.102646,\
	240 0.845561 0.887322 0.099702,\
	241 0.855810 0.888601 0.097452,\
	242 0.866013 0.889868 0.095953,\
	243 0.876168 0.891125 0.095250,\
	244 0.886271 0.892374 0.095374,\
	245 0.896320 0.893616 0.096335,\
	246 0.906311 0.894855 0.098125,\
	247 0.916242 0.896091 0.100717,\
	248 0.926106 0.897330 0.104071,\
	249 0.935904 0.898570 0.108131,\
	250 0.945636 0.899815 0.112838,\
	251 0.955300 0.901065 0.118128,\
	252 0.964894 0.902323 0.123941,\
	253 0.974417 0.903590 0.130215,\
	254 0.983868 0.904867 0.136897,\
	255 0.993248 0.906157 0.143936)
	`)
	fmt.Fprintf(&b, "set cbrange [0:255]\n")
	if maxBytes > 2 {
		fmt.Fprintf(&b, "set cbtics ('0' 0, '%d' 128, '%d' 255)\n", maxBytes/2, maxBytes)
	} else {
		fmt.Fprintf(&b, "set cbtics ('0' 0, '%d' 255)\n", maxBytes)
	}
	return b.String()
}

func (d LinearViridisDistribution) Name() string {
	return "linear_viridis"
}

type LogarithmDistribution struct{}

func (d LogarithmDistribution) Map(value int, maxBytes int) int {
	return value
}

func (d LogarithmDistribution) GnuplotConfig(minBytes int, maxBytes int) string {
	b := strings.Builder{}
	fmt.Fprintf(&b, "set palette defined( 1 'white', %d 'black' )\n", maxBytes)
	fmt.Fprintf(&b, "set logscale cb\n")
	fmt.Fprintf(&b, "set cbrange [1:%d]\n", maxBytes)
	// generate ticks
	fmt.Fprintf(&b, "set cbtics (1")
	for val := 10; val < maxBytes; val *= 10 {
		fmt.Fprintf(&b, ", %d", val)
	}
	fmt.Fprintf(&b, ", %d)\n", maxBytes)
	return b.String()
}

func (d LogarithmDistribution) Name() string {
	return "logarithm"
}

type QuadraticDistribution struct{}

func (d QuadraticDistribution) Map(value int, maxBytes int) int {
	return int(256.0 * math.Sqrt(float64(value)) / math.Sqrt(float64(maxBytes)))
}

func (d QuadraticDistribution) GnuplotConfig(minBytes int, maxBytes int) string {
	b := strings.Builder{}
	fmt.Fprintf(&b, "set palette defined(\\\n")
	for i := 0; i <= 255; i += 1 {
		color := 1.0 - math.Sqrt(float64(i)/256.0)
		if i == 255 {
			fmt.Fprintf(&b, "%d %f %f %f)\n", i, color, color, color)
		} else {
			fmt.Fprintf(&b, "%d %f %f %f,\\\n", i, color, color, color)
		}
	}
	fmt.Fprintf(&b, "set cbrange [0:255]\n")
	// generate ticks
	fmt.Fprintf(&b, "set cbtics ('0' 0, '%d' 64, '%d' 128, '%d' 255)\n", maxBytes/2, int(float64(maxBytes)*math.Sqrt(2)/2.0), maxBytes)
	return b.String()
}

func (d QuadraticDistribution) Name() string {
	return "quadratic"
}

// AllPatterns plots the sum of all patterns found during the post-mortem analysis
func AllPatterns(dir string, outputDir string, numRanks int, patterns map[int]patterns.Data, locationsData []*location.Data) error {
	gnuplotScripts, err := generateAllPatternsDataFiles(dir, outputDir, numRanks, patterns, locationsData)
	if err != nil {
		return fmt.Errorf("generateAllPatternsDataFiles() failed: %s", err)
	}

	for _, gnuplotScript := range gnuplotScripts {
		err = runGnuplot(gnuplotScript, outputDir)
		if err != nil {
			return fmt.Errorf("runGnuplot() failed: %s", err)
		}
	}

	return nil
}
