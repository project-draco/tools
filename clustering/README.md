# Draco Clustering Tool (DCT)

DCT reads a Module Dependency Graph (MDG) from standard input and writes a clustered graph in DOT format on standard output.

## Pre-compiled releases

https://github.com/project-draco/tools/releases/tag/v1.0

## Install from sources

```$ go get -u github.com/project-draco/tools/clustering```

## Running

```$ clustering[.exe|-macos|-linux|-linux-arm] [--multi] [--repeat=n] < software.mdg > software.dot```
