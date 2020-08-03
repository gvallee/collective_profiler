# Copyright (c) 2020 NVIDIA CORPORATION. All rights reserved.

all: convert map profile sendprofiler recvprofiler getcounters validate srcountsanalyzer extracttimings analyzetimings analyzebacktraces getcalldata getbins

convert:
	cd cmd/convert; go build convert.go

map:
	cd cmd/map; go build map.go

profile:
	cd cmd/profile; go build profile.go

getbins:
	cd cmd/getbins; go build getbins.go

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

test:
	cd internal/pkg/grouping; go test -v
	cd internal/pkg/patterns; go test -v
	cd internal/pkg/bins; go test -v
	# cd internal/pkg/maps; go test -v
	cd internal/pkg/notation; go test -v
	cd internal/pkg/counts; go test -v

check: test

clean:
	@rm -f cmd/countersextractor/validate
	@rm -f cmd/getcounters/getcounters
	@rm -f cmd/srcountsanalyzer/srcountsanalyzer
	@rm -f cmd/sendprofiler/sendprofiler
	@rm -f cmd/recvprofiler/recvprofiler
	@rm -f cmd/extracttimings/extracttimings
	@rm -f cmd/analyzetimings/analyzetimings
	@rm -f cmd/analyzebacktraces/analyzebacktraces
	@rm -f cmd/getcalldata/getcalldata
	@rm -f cmd/getbins/getbins
	@rm -f cmd/validate/validate
	@rm -f cmd/profile/profile
	@rm -f cmd/map/map
	@rm -f cmd/convert/convert
