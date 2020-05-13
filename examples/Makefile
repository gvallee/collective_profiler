all: alltoallv_c alltoallv_f

alltoallv_c: ../alltoallv/grouping.o alltoallv.c
	mpicc -g alltoallv.c -o alltoallv_c

alltoallv_f: ../alltoallv/grouping.o alltoallv.f90
	mpifort -g alltoallv.f90 -o alltoallv_f

clean:
	@rm -f alltoallv_c alltoallv_f
