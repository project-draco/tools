Running
==
We have two options:
- docker
  ```
  $ docker run -it --rm -v <absolute path of git repo>:/repo projectdraco/mining-cochange --help
  ```
- build
  (requires go 1.10 or newer)
  ```
  $ go build -o co-change
  $ cd <git repo>
  $ <path to co-change binary> --help
  ```
