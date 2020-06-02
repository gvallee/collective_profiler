# Copyright (c) 2020 NVIDIA CORPORATION. All rights reserved.

all: sendprofiler recvprofiler getcounters validate srcountsanalyzer extracttimings analyzetimings analyzebacktraces getcalldata

getcalldata:
	cd cmd/getcalldata; go build getcalldata.go

analyzebacktraces:
	cd cmd/analyzebacktraces; go build analyzebacktraces.go

analyzetimings:
	cd cmd/analyzetimings; go build analyzetimings.go

validate:
	cd cmd/validate; go build validate.go

getcounters:
	cd cmd/getcounters; go build getcounters.go

srcountsanalyzer:
	cd cmd/srcountsanalyzer; go build srcountsanalyzer.go

sendprofiler:
	cd cmd/sendprofiler; go build sendprofiler.go

recvprofiler:
	cd cmd/recvprofiler; go build recvprofiler.go

extracttimings:
	cd cmd/extracttimings; go build extracttimings.go

clean:
	@rm -f cmd/countersextractor/validate
	@rm -f cmd/getcounters/getcounters
	@rm -f cmd/srcountanalyzer/srcountanalyzer
	@rm -f cmd/sendprofiler/sendprofiler
	@rm -f cmd/recvprofiler/recvprofiler
	@rm -f cmd/extracttimings/extracttimings
	@rm -f cmd/analyzetimings/analyzetimings
	@rm -f cmd/analyzebacktraces/analyzebacktraces
	@rm -f cmd/getcalldata/getcalldata