//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package grouping

import (
	"fmt"
	"log"
)

/*
 * grouping implements an algorithm that grouping data points that are practically a rank and an
 * associated value. In this software package, the value is the amount of data
 * that a rank is sending or receiving.
 * The grouping algorithm is quite simple:
 *  - We compare the median and the mean of the values and if they are too much
 *    appart (10% of the highest value by default), the group is removed and
 *    individual data point put back into the group to the left or the right,
 *    whichever the closer.
 *  - When checking if a group needs to be dismantled, we also check if we would
 *    have a better repartition of the data points by splitting the group in two.
 *  - The algorithm is recursive so when a group is dismantled and data points
 *    added to the group to the left or the right, these groups can end up
 *    behind dismantled too. The aglorithm is supposed to stabilize since a group
 *    can be composed of a single data point.
 */

type Group struct {
	Elts      []int
	Min       int
	Max       int
	CachedSum int
}

type Engine struct {
	Groups []*Group
}

const (
	DEFAULT_MEAN_MEDIAN_DEVIATION = 0.1 // max of 10% of deviation
)

func getRemainder(n int, d int) float64 {
	return float64(n - d*(n/d))
}

func getValue(rank int, values []int) int {
	return values[rank]
}

func getDistanceFromGroup(val int, gp *Group) (int, error) {
	if gp.Max > val && val > gp.Min {
		// something wrong, the value belong to the group
		return -1, fmt.Errorf("value belongs to group")
	}

	if gp.Max <= val {
		return val - gp.Max, nil
	}

	if gp.Min >= val {
		return gp.Min - val, nil
	}

	return -1, nil
}

/**
 * lookupGroup finds the group that is the most likely to accept the data
 * point. For that we scan the min/max of each group, if the value is within
 * the min/max, the group is selected. If the value is between the max of a
 * group and the min of another group, we calculate the distance to each and
 * select the closest group
 */
func (e *Engine) lookupGroup(val int) (*Group, error) {
	index := 0
	if len(e.Groups) == 0 {
		return nil, nil
	}

	log.Printf("Looking up group for value %d", val)
	for _, g := range e.Groups {
		log.Printf("Group #%d, min: %d, max: %d", index, g.Min, g.Max)
		// Within Min and Max of a group
		if g.Min <= val && g.Max >= val {
			return g, nil
		}

		// the value is beyond the last group
		if index == len(e.Groups)-1 && val > g.Max {
			return g, nil
		}

		// the value is before the first group
		if index == 0 && val < g.Min {
			return g, nil
		}

		// the value is in-between 2 groups
		if g.Max < val && index < len(e.Groups)-1 && e.Groups[index+1].Min > val {
			d1, err := getDistanceFromGroup(val, g)
			if err != nil {
				return nil, err
			}
			d2, err := getDistanceFromGroup(val, e.Groups[index+1])
			if err != nil {
				return nil, err
			}

			if d1 <= d2 {
				return g, nil
			}

			return e.Groups[index+1], nil
		}

		index++
	}

	return nil, fmt.Errorf("unable to correctly scan groups")
}

func (gp *Group) addAndShift(rank int, index int) error {
	var newList []int
	newList = append(newList, gp.Elts[:index]...)
	newList = append(newList, rank)
	newList = append(newList, gp.Elts[index:]...)
	gp.Elts = newList
	return nil
}

func (gp *Group) addElt(rank int, values []int) error {
	val := values[rank]
	log.Printf("Adding element %d-%d to group with min=%d and max=%d", rank, val, gp.Min, gp.Max)

	// The array is ordered
	log.Printf("Inserting new element in group's elements")
	i := 0
	// It is not unusual to have the same values coming over and over
	// so we check with the max value of the group, it actually saves
	// time quite often
	if val >= gp.Max {
		i = len(gp.Elts)
	} else {
		for i < len(gp.Elts) && values[gp.Elts[i]] <= values[rank] {
			i++
		}
	}

	if i == len(gp.Elts) {
		// We add the new value at the end of the array
		log.Printf("Inserting element at the end of the group")
		gp.Elts = append(gp.Elts, rank)
	} else {
		log.Printf("Shifting elements within the group at index %d...", i)
		err := gp.addAndShift(rank, i)
		if err != nil {
			return err
		}
	}

	log.Printf("Updating group's metadata (first rank is %d)...", rank)
	//gp.Size++
	gp.CachedSum += values[rank]
	gp.Min = values[gp.Elts[0]]
	gp.Max = values[gp.Elts[len(gp.Elts)-1]]
	log.Printf("Element successfully added (size: %d; min: %d, max: %d)", len(gp.Elts), gp.Min, gp.Max)
	return nil
}

func createGroup(rank int, val int, values []int) (*Group, error) {
	newGroup := new(Group)
	newGroup.Min = val
	newGroup.Max = val
	err := newGroup.addElt(rank, values)
	if err != nil {
		return nil, err
	}

	return newGroup, nil
}

func (e *Engine) addGroup(gp *Group) error {
	index := 0
	if len(e.Groups) == 0 {
		e.Groups = append(e.Groups, gp)
		return nil
	}

	log.Printf("Adding group with min: %d and max: %d", gp.Min, gp.Max)
	for _, g := range e.Groups {
		log.Printf("Comparing with group with min: %d and max: %d", g.Min, g.Max)
		if gp.Min < g.Max {
			break
		}
		index++
	}

	log.Printf("Need to insert new group at index: %d", index)
	return e.insertGroup(gp, index)
}

func getMedian(size int, data []int, values []int) float64 {
	if size == 1 {
		return float64(values[data[0]])
	}

	idx1 := 0
	idx2 := 0
	if getRemainder(size, 2) == 1 {
		idx1 = data[size/2]
		return float64(values[idx1])
	}

	idx1 = size/2 - 1
	idx2 = size / 2
	sum := values[data[idx1]] + values[data[idx2]]
	median := sum / 2
	return float64(median)
}

func (gp *Group) getMedianWithAdditionalPoint(id int, val int, values []int) float64 {
	middle := len(gp.Elts) / 2

	if len(gp.Elts) == 1 {
		log.Println("Only one elements, manually calculating median with new element")
		return (float64(values[gp.Elts[0]]+val) / 2)
	}

	if getRemainder(len(gp.Elts)+1, 2) == 0 {
		// Odd total number of data points, even number of elements already in the group
		// Regardless of where the extra data point would land in the sorted list
		// of the group's elements, the point in the middle of the group's elements
		// will always be used.

		// the two values used to calculate the median
		value1 := -1
		value2 := -1

		index := middle
		candidateRank := gp.Elts[index]
		if values[candidateRank] > val && val > values[candidateRank-1] {
			// The extra element goes in between the middle of the group's elements and the element to its left.
			// It shifts the element to the left to calculate the median
			value1 = val
			value2 = values[candidateRank]
		}
		if values[candidateRank] > val && val < values[candidateRank-1] {
			// The extra element falls toward the begining of the group's elements; it shifts the two elements
			// required to calculate the median
			value1 = values[candidateRank-1]
			value2 = values[candidateRank]
		}
		if values[candidateRank] < val && val < values[candidateRank+1] {
			// The extra element falls in between the middle of the group's elements and the element to its right.
			value1 = values[candidateRank]
			value2 = val
		}
		if values[candidateRank] < val && val > values[candidateRank+1] {
			// The extra element falls toward the end of the group's elements.
			value1 = values[candidateRank]
			value2 = values[candidateRank+1]
		}
		if values[candidateRank] == val {
			// If the extra element has the same value than the middle of the group's elements, it will be added
			// right in the middle, shifting the second half of the group's elements starting by the elements to
			// the left
			value1 = val
			value2 = values[candidateRank]
		}
		if values[candidateRank+1] == val {
			// If the extra elements has the same value then the middle + 1 of the group's elements, it will be
			// added to the right of the middle, shifting the second half of the group's elements starting by the
			// elements to the right
			value1 = val
			value2 = values[candidateRank+1]
		}
		/*
			 if value1 == -1 || value2 == -1 {
				 for i := 0; i < len(gp->Elts); i++ {
					 fprintf(stderr, "-> elt %d: rank: %d, value: %d\n", i, gp->elts[i], values[gp->elts[i]]);
				 }
			 }
		*/

		sum := value1 + value2
		median := float64(sum) / 2
		return median
	} else {
		// Even total number of data points, odd number of elements already in group
		index := middle - 1
		candidateRank := gp.Elts[index]
		if values[candidateRank] > val {
			// The new value falls to the left of the two elements from the original group that are candidate
			return float64(values[candidateRank])
		}
		if values[gp.Elts[index+1]] < val {
			// The new value falls to the right of the two elements from the original group that are candidate
			return float64(values[gp.Elts[index+1]])
		}
		if values[candidateRank] <= val && values[gp.Elts[index+1]] >= val {
			// The new element falls right in the middle of the new group
			return float64(val)
		}
	}

	// We should not actually get here
	return -1
}

func (gp *Group) getMedian(values []int) float64 {
	return getMedian(len(gp.Elts), gp.Elts, values)
}

func (gp *Group) getMean(values []int) float64 {
	log.Printf("Calculating mean based on cache sum: %d and %d elements", gp.CachedSum, len(gp.Elts))
	return float64(gp.CachedSum / len(gp.Elts))
}

func affinityIsOkay(mean float64, median float64) bool {
	// If the mean and median do not deviate too much, we add the new data point to the group
	// Once the new data point is added to the group, we check the group to see if it needs
	// to be split.
	maxMeanMedian := float64(0)
	minMeanMedian := float64(0)
	affinityOkay := false // true when the mean and median are in acceptable range
	if median > mean {
		maxMeanMedian = median
		minMeanMedian = mean
	} else {
		maxMeanMedian = mean
		minMeanMedian = median
	}

	log.Printf("Mean: %f; median: %f", mean, median)
	a := maxMeanMedian * (1 - DEFAULT_MEAN_MEDIAN_DEVIATION)
	if a <= minMeanMedian {
		affinityOkay = true
	}

	return affinityOkay
}

func (gp *Group) groupIsBalanced(values []int) bool {
	// We calculate the mean and median.
	median := gp.getMedian(values)
	mean := gp.getMean(values)

	return affinityIsOkay(mean, median)
}

func (e *Engine) unlinkGroup(gp *Group) error {
	index := 0
	for _, g := range e.Groups {
		if g == gp {
			break
		}
		index++
	}

	if index >= len(e.Groups) {
		return fmt.Errorf("cannot find group")
	}

	// we must be careful to keep the order.
	e.Groups = append(e.Groups[:index], e.Groups[index+1:]...)

	return nil
}

func groupToString(values []int) string {
	str := ""
	for _, v := range values {
		str = fmt.Sprintf("%s %d", str, v)
	}
	return str
}

func (e *Engine) insertGroup(gp *Group, index int) error {
	log.Printf("Inserting group at index: %d", index)
	var newGroupList []*Group
	if index == 0 {
		e.Groups = append([]*Group{gp}, e.Groups...)
	} else {
		newGroupList = append(newGroupList, e.Groups[:index]...)
		newGroupList = append(newGroupList, gp)
		e.Groups = append(newGroupList, e.Groups[index:]...)
	}
	return nil
}

func (e *Engine) splitGroup(gp *Group, indexSplit int, values []int) (*Group, error) {
	// Create the new group
	ng, err := createGroup(gp.Elts[indexSplit], values[gp.Elts[indexSplit]], values)
	if err != nil {
		return nil, err
	}

	// Find index of the group
	i := 0
	for i < len(e.Groups) {
		if e.Groups[i] == gp {
			break
		}
		i++
	}

	if i == len(e.Groups) {
		return nil, fmt.Errorf("unable to find group")
	}

	log.Printf("group index is: %d", i)

	// Transfer all the elements to transfer into the new group into a temporary list
	// We do not include the element at indexSplit because it is already in the new
	// group
	var temp []int
	for j := indexSplit + 1; j < len(gp.Elts); j++ {
		temp = append(temp, e.Groups[i].Elts[j])
	}

	// Remove all the elements that are moving to the new group
	e.Groups[i].Elts = e.Groups[i].Elts[:indexSplit]
	log.Printf("Split group now has %d elements", len(e.Groups[i].Elts))

	// Update the group's metadata after removal of elements
	e.Groups[i].CachedSum = 0
	for j := 0; j < len(e.Groups[i].Elts); j++ {
		e.Groups[i].CachedSum += values[e.Groups[i].Elts[j]]
	}
	gp.Min = values[gp.Elts[0]]
	gp.Max = values[gp.Elts[len(gp.Elts)-1]]

	// Transfer elements from initial group to new one
	for j := 0; j < len(temp); j++ {
		err := ng.addElt(temp[j], values)
		if err != nil {
			return nil, err
		}
	}

	// Finally we add the new group
	log.Printf("Group split, inserting new group at index %d", i+1)
	err = e.insertGroup(ng, i+1)
	if err != nil {
		return nil, err
	}

	return ng, nil
}

func (e *Engine) balanceGroupWithNewElement(gp *Group, id int, val int, values []int) error {
	sum := float64(gp.CachedSum + val)
	mean := float64(sum / float64(len(gp.Elts)+1))
	log.Printf("Mean of %d with %d elements is %f", gp.CachedSum+val, len(gp.Elts)+1, mean)

	// Now we calculate the median
	median := gp.getMedianWithAdditionalPoint(id, val, values)
	if affinityIsOkay(mean, median) {
		err := gp.addElt(id, values)
		if err != nil {
			return err
		}
	} else {
		log.Println("Group needs to be split")
		// We figure out where we need to split the group
		i := 0
		for i < len(gp.Elts) && values[gp.Elts[i]] < values[id] {
			i++
		}

		if i < len(gp.Elts) {
			log.Printf("Group needs to split in two")
			newGroup, err := e.splitGroup(gp, i, values)
			if err != nil {
				return err
			}
			// We find the group that is the closest to the element to add
			d1, err := getDistanceFromGroup(values[id], gp)
			if err != nil {
				return err
			}
			d2, err := getDistanceFromGroup(values[id], newGroup)
			if err != nil {
				return err
			}
			if d2 < d1 {
				err := newGroup.addElt(id, values)
				if err != nil {
					return err
				}
			} else {
				err := gp.addElt(id, values)
				if err != nil {
					return err
				}
			}
		} else {
			log.Printf("Group spliting only needs to add new group at the end (index: %d, len: %d)", i, len(gp.Elts))
			newGroup, err := createGroup(id, val, values)
			if err != nil {
				return err
			}
			err = e.addGroup(newGroup)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) AddDatapoint(id int, values []int) error {
	val := getValue(id, values)

	// We scan the groups to see which group is the most likely to be suitable
	gp, err := e.lookupGroup(val)
	if err != nil {
		return err
	}
	if gp == nil {
		log.Println("No group found, creating a new one")
		gp, err := createGroup(id, val, values)
		if err != nil {
			return nil
		}
		err = e.addGroup(gp)
		if err != nil {
			return err
		}
	} else {
		log.Println("Adding data point to existing group")
		err := e.balanceGroupWithNewElement(gp, id, val, values)
		if err != nil {
			return err
		}
	}

	return nil
}

func Init() *Engine {
	newEngine := new(Engine)
	return newEngine
}

/*
func (e *Engine) Fini() error {
	// Anything to do?
	return nil
}
*/

func (e *Engine) GetGroups() ([]*Group, error) {
	return e.Groups, nil
}
