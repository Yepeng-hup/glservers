#!/bin/bash

path=/all/go-project/glservers

cd ${path} && exec nohup ./glserver >& ./log/glserver.log &
