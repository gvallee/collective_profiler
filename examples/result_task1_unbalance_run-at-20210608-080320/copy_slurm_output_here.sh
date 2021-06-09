#!/bin/bash
RESULTS_ROOT=$(dirname "$0")
# source "$RESULTS_ROOT/slurm_variable.log" # does not work because file has illegal values for bash variables
eval $( grep SLURM_SUBMIT_DIR "$RESULTS_ROOT/slurm_variable.log" )
eval $( grep SLURM_JOB_NAME  "$RESULTS_ROOT/slurm_variable.log" )
eval $( grep SLURM_JOB_ID  "$RESULTS_ROOT/slurm_variable.log" )
SLURM_OUTPUT_FILE=$(ls "$SLURM_SUBMIT_DIR" | grep "$SLURM_JOB_NAME" | grep "$SLURM_JOB_ID")
echo "copying $SLURM_SUBMIT_DIR/$SLURM_OUTPUT_FILE here ..."
cp "$SLURM_SUBMIT_DIR/$SLURM_OUTPUT_FILE" "$RESULTS_ROOT"
chmod a=r "$RESULTS_ROOT/$SLURM_OUTPUT_FILE"
