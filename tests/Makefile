all: liballtoallv_dummy.so

liballtoallv_dummy.so: dummy_pmpi.c
	mpicc -g -shared -fPIC dummy_pmpi.c -o liballtoallv_dummy.so

clean:
	@rm -f *.so *.o