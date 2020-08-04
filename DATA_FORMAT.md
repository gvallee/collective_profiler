# Introduction

Alltoallv operations include send and receive counts, which are arrays of integers.
Practically, it means that all rank involved in a alltoallv operation specify the
count of element of a given type to send to a destination rank.
For example, if rank 0 has the following send counts: 0 0 1 2; it means that
rank 0 sends 0 element to rank 0 and 1, 1 element to rank 2 and 2 elements to rank 3.
It also means that the size of the communicator used for the alltoallv operation is 4.

# Counts profiling

Based on the introduction of alltoallv operations, the profiler forces rank 0 of the
communicator used to perform the alltoallv operation to gather the send and receive
counts of all the ranks involved in the operation. If we note the size of the
communicator as N, rank 0 of the communicator therefore create a NxN matrix, where lines
are the ranks defining the counts and the columns the destination rank.
To illustrate this, let's consider the following send count matrix:
```
1 2 
3 4
```
This means that rank 9 is sending 1 element to rank 0 and 2 elements to rank 1; 
while rank 1 is sending 3 elements to rank 0 and 4 elements to rank 1.
A more explicit way represent the matrix could be:
```
rank 0: 1 2
rank 1: 3 4
```

## Non-compact count files

The profiler has the capability of saving the counts of all alltoallv operations.
The file is names 'counts-rankX_callY.md', where X is the rank in COMMWORLD of the
rank 0 of the communicator used for the operation (also called lead rank), and Y
is the call number for which the rank has been involved in.
This type of files provides:
- the size of the communicator
- the size of the send datatype
- the size of the receive datatype
- the send counts
- the receive counts

This format is well-suited for systems where the application requires all the memory available
and for which the file system can handle a large number of files (a file is created for each
alltoallv call).

## Compact count files

It is obvious that for large applications, the non-compact format may end up creating
a lot of file and represent overall a large amount of data, making it more difficult to
analyze the data. To address this issues, the profiler also supports a compact format.
This format makes the count matrix more compact: if two ranks have the same counts, they
will be aggregated and some meta-data is created to track which ranks have these counts.
For instance, if rank 0 and 1024 have the same counts, the following line will be in
the matrix:
```
Rank(s) 0,1024: 0 0 0 1 1 1 1 0 0 0 0
```
If rank 1 to 5 and 1024 have the same counts, the following line will be in the matrix:
```
Rank(s) 1-5,1024: : 0 0 0 1 1 1 1 0 0 0 0
```
The profiler performs the same type of data compression across calls: if two calls have
the exact same counts and datatype size, the metadata is just updated to track the calls
associated to the counts:
```
# Raw counters
  
Number of ranks: 3
Datatype size: 8
Alltoallv calls  0-2
Count: 2 calls - 0-1


BEGINNING DATA
Rank(s) 0: 1 2 0
Rank(s) 1: 0 0 3
Rank(s) 2: 1 0 0
END DATA
```

# Patterns

Parsing the send and receive counts, it is possible to detect patterns. Patterns are
data about how many ranks receive data from or send data to other ranks.
Let's consider the following send matrix:
```
Rank(s) 0-4: 1 1 1 0
```
The pattern is '4 ranks are send 3 other ranks'.