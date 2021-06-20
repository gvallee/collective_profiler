# What we have changed?

In the modifications of the code, we have added the required functions and variables that we have needed to implement the Task 3 and Task 4 of the Coding Challenge.

We have tried to follow the basis of the profiler structure, in order to follow an easy integration and update of future codes.

We have also followed the way of generating the diagrams, using gnuplot, following the same pattern of generating gnuplots as the current profiler.

# Changes made for Task 3

## Generating heatmaps

For the creation of the heatmaps for the different patterns, in the function `Analyze` of the `profiler.go` file we have added some function calls at STEP 6.

- The function `GetSendDataForTask3` gets the array of patterns of bytes sent from the sender ranks to the receiver ranks. These data are obtained from the `counts_reader.go` file, inside the function `LoadCallsData`, where we append the `sendCounts` map to the array of patterns in each iteration. If we have more than one communicator, this array will have all patterns of all communicators.

- As we only want the first patterns of the first communicator, with the function GetNumberSendDataForTask3(), where we get the number of the patterns in one communicator. Next we create an slice with only the patterns we want.

- With the function `GetNumberOfCalls` we get an array with the proportion of pattern calls over the total number of calls. To get it, inside the function we read the file `send-counters.job0.rank0.txt`, and we keep only the strings with the proportions (e.g., 121/964).

- Finally we have the call `Task3`, which generates the plots using the returns of the two previous functions.
This function, located at `plot.go`, iterates over the patterns, and in every iteration generates a matrix ready to be plotted, using the original matrix of bytes sent between ranks.
At the end of every iteration, generates the gnuplot.

## Visualization in the WebUI

To show the gnuplots in the WebUI, we have added a tab called Task3, with the same layout as the Calls tab.

We have only edited the files `webui.go` and `index.html`, and added the files `task3Details.html` and `task3Layout`.

Inside the files we have "replicated" the functions and variables of the call and calls data. The major difference is in the function `serviceTask3HeatmapDetailsRequest`, where we only watch for the param that indicates the number of pattern.

The two added files `task3Details.html` and `task3Layout` are also very similar to the `callDetails.html` and `callsLayout` files.

# Changes made for Task 4

## Generating heatmaps

To generate the weighted sum of all patterns, we have added a call to `Task4` in the file `profiler.go`, following the `Task3` call.

All the steps are almost the same as the ones we have done for task 3.

Inside `Task4` function we first calculate the sum of all pattern cells for every cell, and then we proceed to generate the plot.

## Visualization in the WebUI

For the visualization of the generated gnuplot in the WebUI we have followed the same steps as for the task3, although now we only show one plot in the `Task4` tab.
