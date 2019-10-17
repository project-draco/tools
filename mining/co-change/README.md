Running
==
We have two options:
- docker
  ```
  $ docker build -t co-change .
  $ docker run -it --rm -v <absolute path of git repo>:/repo co-change [<string to ignore>]
  ```
- build
  (requires go 1.10 or newer)
  ```
  $ go build -o co-change
  $ cd <git repo>
  $ <path to co-change binary> --help
  ```
