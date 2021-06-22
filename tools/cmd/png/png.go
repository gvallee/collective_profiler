package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const max_ranks = 200
const max_patterns = 30
const width = 10

var ranks int

type TestStringList []string

//元素个数
func (t TestStringList) Len() int {
	return len(t)
}

//比较结果
func (t TestStringList) Less(i, j int) bool {
	iw := weight_from_pattern(t[i])
	jw := weight_from_pattern(t[j])
	return iw > jw
}

//交换方式
func (t TestStringList) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

//从pattern原始文本提取weight
func weight_from_pattern(str string) int {
	res, _ := strconv.Atoi(strings.Split(strings.Split(str, "Count: ")[1], " calls")[0])
	return res
}

//pattern转rgb数组
func pattern_to_sz(str string) [max_ranks][max_ranks]int {
	var res [max_ranks][max_ranks]int
	totalRank, _ := strconv.Atoi(strings.Split(strings.Split(str, "Number of ranks: ")[1], "\n")[0])
	ranks = totalRank
	items := strings.Split(str, "Rank(s) ")
	items = items[1:]
	for _, item := range items {
		part := strings.Split(item, ":")
		tmp := strings.Split(part[0], ",")
		var from [max_ranks]bool
		for i := 0; i < totalRank; i++ {
			from[i] = false
		}
		for _, x := range tmp {
			if strings.Contains(x, "-") {
				splits := strings.Split(x, "-")
				tmpi, _ := strconv.Atoi(splits[0])
				tmpj, _ := strconv.Atoi(splits[1])
				for i := tmpi; i <= tmpj; i++ {
					from[i] = true
				}
			} else {
				tmpi, _ := strconv.Atoi(x)
				from[tmpi] = true
			}
		}
		dest := strings.Fields(part[1])
		for i := 0; i < totalRank; i++ {
			if from[i] {
				for j, x := range dest {
					tmpx, _ := strconv.Atoi(x)
					res[i][j] += tmpx
				}
			}
		}
	}
	return res
}

//数据量转rgb值，task3
func num_to_rgb_8color(num int) [3]uint8 {
	if num == 0 {
		return [3]uint8{255, 255, 255} //white
	} else if num <= 10 {
		return [3]uint8{255, 255, 0} //yellow
	} else if num <= 100 {
		return [3]uint8{255, 165, 0} //orange
	} else if num <= 1000 {
		return [3]uint8{0, 255, 0} //green
	} else if num <= 10000 {
		return [3]uint8{255, 0, 0} //red
	} else if num <= 100000 {
		return [3]uint8{160, 32, 240} //purple
	} else if num <= 1000000 {
		return [3]uint8{165, 42, 42} //brown
	} else {
		return [3]uint8{0, 0, 0} //black
	}
}

//输出不大于的num的最大的10的n次幂
func num_to_low(num int) int {
	times := int(math.Log10(float64(num)))
	res := 1
	for i := 0; i < times; i++ {
		res *= 10
	}
	return res
}

//数据量转rgb值，task5的第一种方法（线性）
func num_to_rgb_linear(num int) [3]uint8 {
	depth := num / 4000
	if depth > 255 {
		depth = 255
	}
	depth = 255 - depth
	return [3]uint8{uint8(depth), uint8(depth), uint8(depth)}
}

//数据量转rgb值，task5的第二种方法（对数）
func num_to_rgb_Logarithmic(num int) [3]uint8 {
	depth := math.Log10(float64(num+1)) * 42.66
	if depth > 255 {
		depth = 255
	}
	depth = 255 - depth
	return [3]uint8{uint8(depth), uint8(depth), uint8(depth)}
}

//数据量转rgb值，task5的第三种方法（自己的）
func num_to_rgb_own(num int) [3]uint8 {
	low := num_to_low(num)
	high := low * 10
	if low == 1 {
		low = 0
	}
	lowrgb := num_to_rgb_8color(low)
	highrgb := num_to_rgb_8color(high)

	var resrgb [3]uint8
	for i := 0; i < 3; i++ {
		resrgb[i] = uint8((float64(lowrgb[i])*float64(high-num) + float64(highrgb[i])*float64(num-low)) / float64(high-low))
	}
	return resrgb
}

//rgb数组转png图像
func sz_to_png(sz [max_ranks][max_ranks]int, path string, mode int) {
	file, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	rgba := image.NewRGBA(image.Rect(0, 0, ranks*width, ranks*width))
	for x := 0; x < ranks*width; x++ {
		for y := 0; y < ranks*width; y++ {
			var rgb [3]uint8
			if mode == 0 {
				rgb = num_to_rgb_8color(sz[x/width][y/width])
			} else if mode == 1 {
				rgb = num_to_rgb_linear(sz[x/width][y/width])
			} else if mode == 2 {
				rgb = num_to_rgb_Logarithmic(sz[x/width][y/width])
			} else if mode == 3 {
				rgb = num_to_rgb_own(sz[x/width][y/width])
			}

			rgba.Set(x, y, color.RGBA{rgb[0], rgb[1], rgb[2], 255})
		}
	}
	err = png.Encode(file, rgba)
	if err != nil {
		fmt.Println(err)
	}
}
func main() {
	var filein string
	var filepath string
	if len(os.Args) > 1 {
		filein = os.Args[1]
		lastpiepos := strings.LastIndex(filein, "/")
		if lastpiepos == -1 {
			fmt.Println("Can't guess path of input file")
			os.Exit(0)
		}
		filepath = filein[:lastpiepos]
	} else {
		fmt.Println("No input file!")
		return
	}
	data, err := ioutil.ReadFile(filein) //输入文件路径
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}
	//截取pattern并排序
	pattern := strings.Split(string(data)[1:], "#")
	sort.Sort(TestStringList(pattern))
	//输出每个pattern的weight到weight.txt
	outputFile, outputError := os.OpenFile(filepath+"/weight.txt", os.O_WRONLY|os.O_CREATE, 0666)
	if outputError != nil {
		fmt.Println(outputError)
		return
	}
	defer outputFile.Close()
	outputWriter := bufio.NewWriter(outputFile)
	var weight [max_patterns]int
	for i := 0; i < len(pattern); i++ {
		weight[i] = weight_from_pattern(pattern[i])
		outputWriter.WriteString(strconv.Itoa(i) + " " + strconv.Itoa(weight[i]) + "\n")
	}
	outputWriter.Flush()
	//计算rgb数组的加权和，
	var allsz [max_ranks][max_ranks]int
	for i := 0; i < len(pattern); i++ {
		if i > 10 {
			break
		}
		sz := pattern_to_sz(pattern[i])
		for x := 0; x < max_ranks; x++ {
			for y := 0; y < max_ranks; y++ {
				allsz[x][y] += sz[x][y] * weight[i]
			}
		}

		sz_to_png(sz, filepath+"/"+strconv.Itoa(i)+"_task3.png", 0)
		sz_to_png(sz, filepath+"/"+strconv.Itoa(i)+"_task5_linear.png", 1)
		sz_to_png(sz, filepath+"/"+strconv.Itoa(i)+"_task5_log.png", 2)
		sz_to_png(sz, filepath+"/"+strconv.Itoa(i)+"_task5_own.png", 3)
	}
	sz_to_png(allsz, filepath+"/task4.png", 0)
	sz_to_png(allsz, filepath+"/task4_linear.png", 1)
	sz_to_png(allsz, filepath+"/task4_log.png", 2)
	sz_to_png(allsz, filepath+"/task4_own.png", 3)

}
