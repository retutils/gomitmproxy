#!/bin/bash
set -e

echo "Running benchmarks..."
go test -run=NONE -bench=. -benchmem ./proxy ./addon | tee bench_result.txt
