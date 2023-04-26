#!/bin/bash
set -e

influx bucket create -n hammerBucketDetailed -o ddosify
influx bucket create -n hammerBucketIteration -o ddosify
