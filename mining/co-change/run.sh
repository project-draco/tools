#!/bin/sh
if [ "$#" -lt 2 ]; then
        args=''
else
        args=`echo $@ | cut -d ' ' -f -$(($#-1))` # all args but last
fi
eval lastarg=\${$#}
cd / \
&& if [ ! -e /repo ]; then
    git clone --quiet $lastarg repo \
    && cd repo \
    && /co-change $args
else
    cd repo \
    && /co-change $@
fi
