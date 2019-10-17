# Instructions

## Initial setup
- Install [Docker](http://www.docker.com/products/overview)
- Add your SSH key into your git hosting site (GitHub, GitLab, BitBucket, etc.)
- If the repositories to be converted are large,
maybe it will be necessary to add more memory in Docker configuration.

## To convert a repository, follow these steps:
- For remote repositories, create the destination git repository and run the following command in a terminal
  ```
  $ docker run --rm -v $HOME/.ssh:/root/.ssh projectdraco/g2h converter.sh <source-url> <destination-url>
  ```
  The destination-url must use SSH protocol, otherwise a prompt for user name and password will be shown.

  If it is not possible (or wanted) to use SSH, execute bash in docker and run the conversion inside the container, accordingly the following commands.
  ```
  $ docker run -it --rm projectdraco/g2h bash
  # converter.sh <source-url> <destination-url>
  # exit
  ```
- For local repositories, both source and destination, run the following command in a terminal

  First, make sure that the destination repository is initialized as follows
  ```
  $ cd <destination-folder>
  $ git init --bare
  ```
  Then, execute:
  ```
  $ docker run --rm -v <source-folder>:/source -v <destination-folder>:/dest projectdraco/g2h converter.sh /source /dest
  ```
## Build the image (not recommended)
If you want to build a local copy of the image, follow these steps:
- Clone this repository
- In the repository working directory, run the following command in a terminal:
  ```
  $ docker build -t projectdraco/g2h dockerfiles/kenja
  ```
