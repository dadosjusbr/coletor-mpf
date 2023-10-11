#!/bin/bash

months=("01" "02" "03" "04" "05" "06" "07" "08" "09" "10" "11" "12")

for year in {2019..2022}; do
    for month in ${months[*]}; do
        MONTH=${month} YEAR=${year} OUTPUT_FOLDER=membros-inativos-${year} go run . 
    done
done