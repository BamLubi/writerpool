package writerpool

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"
)

var (
	// dataPath is the directory of saved data
	dataPath string = "data"

	// writerBufferSize controls the writer buffer size.
	// The data will be flushed to disk directly if the buffer is full.
	writerBufferSize int = 4096

	// flushThreshold is the manually set flush frequency.
	// Before flush the data to disk, we will check whether the used of buffer is
	// greater than the flushThreshold. If true, we wll first flush the old data
	// in buffer to disk and then write the new data to buffer.
	flushThreshold int = 2048

	// timerFlushDur controls the flush frequency in case of some small data will
	// not trigger the flushThreshold and will always stay in buffer.
	timerFlushDur time.Duration = time.Second * 30

	// timerDeleteDur controls the frequency of deleting unused file and writer pointer.
	// Befor close the file, we will first check whether the buffer has data.
	timerDeleteDur time.Duration = time.Hour
)

type WriterPool struct {
	file        map[string]*os.File
	writer      sync.Map
	bufferMutex sync.Map
	mu          sync.Mutex
}

// New function will return a new object pointer of WriterPool
func New() *WriterPool {
	return &WriterPool{}
}

// Close is used to close all file interface in WriterPool
func (wp *WriterPool) Close() error {
	wp.mu.Lock()

	// flush all data
	wp.writer.Range(func(key, value interface{}) bool {
		writer, ok := value.(*bufio.Writer)
		value, _ = wp.bufferMutex.Load(key.(string))
		mu, _ := value.(*sync.Mutex)
		if ok {
			mu.Lock()
			writer.Flush()
			mu.Unlock()
			return true
		}
		return false
	})

	// close file
	for _, f := range wp.file {
		if err := f.Close(); err != nil {
			wp.mu.Unlock()
			return err
		}
	}

	wp.mu.Unlock()

	return nil
}

// getOperator is used to get the *bufio.Writer and *sync.Mutex of target writer and buffer.
// if the file directory is not exist, it will create the new directory.
// The file path is like "path/to/project/data/{directory}/{filename}.txt".
func (wp *WriterPool) getOperator(dirName string, fileName string) (*bufio.Writer, *sync.Mutex, error) {
	key := fmt.Sprintf("%s.%s", dirName, fileName)

	// set lock to create new file and writer when not exist
	// the lock must set before the first time of writer.Load()
	// because other goroutine will create file concurrently.
	wp.mu.Lock()
	_, ok := wp.writer.Load(key)

	if !ok {
		// if the path is not exist, create the directory
		dir := fmt.Sprintf("%s/%s", dataPath, dirName)
		if err := os.MkdirAll(dir, fs.ModePerm); err != nil {
			wp.mu.Unlock()
			return nil, nil, err
		}

		// create new file
		filePath := fmt.Sprintf("%s/%s.txt", dir, fileName)
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			wp.mu.Unlock()
			return nil, nil, err
		}
		// lazy create file map
		if wp.file == nil {
			wp.file = make(map[string]*os.File)
		}
		wp.file[key] = file

		// create new writer
		w := bufio.NewWriterSize(file, writerBufferSize)
		wp.writer.Store(key, w)

		// create writer lock
		m := &sync.Mutex{}
		wp.bufferMutex.Store(key, m)

		// set a timer to close file
		go wp.SetTimer2Flush(key, w, m)

		// set a timer to delete interface{}
		go wp.SetTimer2Del(key, file, w, m)
	}
	wp.mu.Unlock()

	// get writer
	value, _ := wp.writer.Load(key)
	writer, ok := value.(*bufio.Writer)
	if !ok {
		return nil, nil, fmt.Errorf("get_writer_type_assertion_error with key %s\n", key)
	}

	// get bufferMutex
	value, _ = wp.bufferMutex.Load(key)
	mu, ok := value.(*sync.Mutex)
	if !ok {
		return nil, nil, fmt.Errorf("get_bufferMutex_type_assertion_error with key %s\n", key)
	}

	return writer, mu, nil
}

// SetTimer2Flush is used to Flush Data to disk
// it will make sure that all data can be flushed into file
func (wp *WriterPool) SetTimer2Flush(key string, writer *bufio.Writer, mu *sync.Mutex) {
	timer := time.NewTimer(timerFlushDur)
	defer func() {
		timer.Stop()
		if recover() != nil {
			fmt.Println("timer_flush_panic with key ", key, ", error ", recover())
		}
	}()
	// wait
	<-timer.C
	// flush data
	mu.Lock()
	if writer.Buffered() > 0 {
		writer.Flush()
	}
	mu.Unlock()
}

// SetTimer2Del is used to delete unused interface{}
func (wp *WriterPool) SetTimer2Del(key string, file *os.File, writer *bufio.Writer, mu *sync.Mutex) {
	timer := time.NewTimer(timerDeleteDur)
	defer func() {
		timer.Stop()
		if recover() != nil {
			fmt.Println("timer_del_panic with key ", key, ", error ", recover())
		}
	}()
	// wait
	<-timer.C
	// delete interface{}
	wp.mu.Lock()
	mu.Lock()
	if writer.Buffered() > 0 {
		writer.Flush()
	}
	wp.writer.Delete(key)
	file.Close()
	delete(wp.file, key)
	mu.Unlock()
	wp.bufferMutex.Delete(key)
	wp.mu.Unlock()
}

// WriteStringWithDate will use target date to stroe file
// The file will store at path/to/data/date/hour.txt
func (wp *WriterPool) WriteStringWithDate(dirName string, fileName string, data string) error {
	// get writer
	writer, mu, err := wp.getOperator(dirName, fileName)
	if err != nil {
		return err
	}

	// write data to buffer
	// need to set lock for writer.WriteString
	mu.Lock()
	if writer.Buffered() > flushThreshold {
		writer.Flush()
	}
	if _, err := writer.WriteString(data); err != nil {
		mu.Unlock()
		return err
	}
	mu.Unlock()

	return nil
}

// GetStatus provides a interface to watch the status of writerpool
func (wp *WriterPool) GetStatus() map[string]interface{} {
	var data = map[string]interface{}{
		"dataPath":         dataPath,
		"writerBufferSize": writerBufferSize,
		"flushThreshold":   flushThreshold,
		"timerFlushDur":    timerFlushDur,
		"timerDeleteDur":   timerDeleteDur,
		"openFileNum":      len(wp.file),
	}
	return data
}
