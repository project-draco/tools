#!/bin/bash
cut -d ' ' -f 9 run3.sh | cut -d '/' -f 2 | sed s/\.git// | xargs -I _ sh -c "printf '\n%s,' _ && curl -s https://github.com/project-draco-hr/_ | pup 'li.commits span text{}' | tr -d '[:space:],'" | tee status.txt
# cut -d ' ' -f 10 run.sh | tail -n 120 | cut -d '/' -f 2 | sed s/\.git// | xargs -I _ sh -c "printf '\n%s,' _ && curl -s https://github.com/project-draco-hr/_ | pup 'table.files tr td.content text{}' | sed '/^[[:space:]]*$/d' | wc -l | sed '/^[[:space:]]*$/d'" | tee status.txt
# cut -d ' ' -f 10 run.sh | tail -n 120 | cut -d '/' -f 2 | sed s/\.git// | xargs -I _ sh -c "printf '\n%s,' _ && curl -s https://raw.githubusercontent.com/project-draco-hr/_/master/log.txt | grep completed | tr -d '[:space:]'" | tee status.txt
