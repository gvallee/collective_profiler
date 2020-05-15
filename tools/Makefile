# Copyright (c) 2020 NVIDIA CORPORATION. All rights reserved.

all: countersextractor sendprofiler

countersextractor:
	cd cmd/countersextractor; go build countersextractor.go

sendprofiler:
	cd cmd/sendprofiler; go build sendprofiler.go

clean:
	@rm -f cmd/countersextractor/countersextractor