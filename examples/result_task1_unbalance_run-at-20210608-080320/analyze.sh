#!/bin/bash
# somewhere to keep the results of post processing the results of the cluster job
export RESULTS_ROOT=$( dirname $(readlink --canonicalize --no-newline "$0" ) )
export POST_ANALYSYS_ROOT="$RESULTS_DIR/post_processed"
echo "this script is as yet a dummy and has set only some paths - no analysis performed
# TODO call some post processing scripts
# TODO copy the post processing scripts to the post processing directory for a record copy
# TODO set results of postprocessing to read only
# TODO test all this including the exports above
