# writerpool

file writer pool for large discrete file write.

## Introduction

The main contribute of this project is to develop the writerpool. It will great impove the ability of huge file storage, such as log file store and analysis.

In this project, the log data will be saved to differernt files by hours, and it supports the discrete time logs, that is the arrival time of log will not be sequential. The main designs are shown bellow

- We use `bufio` to save data to disk by a buffer. The data will first be saved to buffer, and flushed to disk if the buffer capacity is less than the threshold.
- We set timer to flush data and delete unused file pointer. The first timer to flush data is acted as a last resort. The second timer is used to save the memory.
- We use `synv.Map` to get file pointer and writer pointer, which is thread safe.

Compare with using `file.WriteString` to save data, our method increased QPS by 5 times (up to 1652 QPS) and only use 38% memory. The details is shown in `writerpool_test.go`.

## Quick Start

```shell
# get the package
go get github.com/BamLubi/writerpool
```

```go
func main() {
    wp := writerpool.New()
    defer wp.Close()

    err := wp.WriteStringWithDate("directory name", "file name", "data")
    if err != nil {
        paninc(err)
    }
}

```