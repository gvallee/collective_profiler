all: alltoallv_c alltoallv_multicomms_c alltoallv_f

alltoallv_c: alltoallv.c
	mpicc -g alltoallv.c -o alltoallv_c

alltoallv_multicomms_c: alltoallv_multicomms.c
	mpicc -g alltoallv_multicomms.c -o alltoallv_multicomms_c

alltoallv_f: alltoallv.f90
	mpifort -g alltoallv.f90 -o alltoallv_f

clean:
	@rm -f alltoallv_c alltoallv_f alltoallv_multicomms_c
