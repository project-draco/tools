#!/bin/bash

if [ ! -e /orig ]; then git clone $1 /orig; fi \
    && cd /orig \
    && git branch -r | grep -v "\->" > /branches \
    && for remote in `cat /branches`; do git branch --track $remote; done \
    && time kenja.historage.convert --logging-level=INFO /orig /hr 2>&1 | tee /log.txt \
    && cd /hr \
    && cp /log.txt . \
    && git add log.txt 2>&1 | tee -a /log.txt \
    && git commit -m 'Add conversion log file' 2>&1 | tee -a /log.txt \
    && if [ ! -z "$2" ]; then
        git remote add origin $2 \
        && git push -u origin master 2>&1 | tee -a /log.txt
       fi
