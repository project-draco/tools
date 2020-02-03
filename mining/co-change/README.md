Running
==
We have two options:
- docker
  ```
  $ docker run -it --rm -v <absolute path of git repo>:/repo projectdraco/mining-cochange --help
  ```
- install
  (requires go 1.10 or newer)
  ```
  $ go get -u github.com/project-draco/tools/mining/co-change
  $ co-change --help
  ```
