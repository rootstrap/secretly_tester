package interleaver

import "io"
import "bufio"
import "sync"
import "os"

type Interleaver struct {
	sink io.Writer
	lock sync.Mutex
}

func NewInterleaver(sink io.Writer) *Interleaver {
	return &Interleaver{sink: sink}
}

func (inter *Interleaver) Copy(source io.Reader) error {
	reader := bufio.NewReader(source)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		inter.lock.Lock()
		_, err = inter.sink.Write(line)
		inter.lock.Unlock()
		if err != nil {
			return err
		}
	}
}

var Stdout = NewInterleaver(os.Stdout)
var Stderr = NewInterleaver(os.Stderr)
