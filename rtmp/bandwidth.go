package rtmp

import (
	"bufio"
	"bytes"
	"os/exec"
	"regexp"
	"strconv"
)

// TODO: figure out buffer sizes

type rtmpTest struct {
	url      string
	cmd      *exec.Cmd
	Progress chan RTMPProgress
}

func NewRTMPTest(rtmpUrl string) *rtmpTest {
	test := new(rtmpTest)
	test.url = rtmpUrl
	test.Progress = make(chan RTMPProgress)
	return test
}

func (t *rtmpTest) Run() error {
	rtmpdumpPath, err := exec.LookPath("rtmpdump")
	if err != nil {
		return err
	}
	t.cmd = exec.Command(rtmpdumpPath, "--realtime", "-r", t.url)
	t.cmd.Stdout = nil
	stream, err := t.cmd.StderrPipe()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(stream)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, nil, nil
		}
		if i := bytes.IndexByte(data, '\r'); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	go func() {
		for scanner.Scan() {
			var progress RTMPProgress
			if parseProgress(scanner.Text(), &progress) {
				t.Progress <- progress
			}
		}
	}()
	return t.cmd.Run()
}

type RTMPProgress struct {
	Seconds, KiloBytes, Percent float32
}

var progressRegexp *regexp.Regexp = regexp.MustCompile("(\\d+[.]\\d+) *kB +/ +(\\d+[.]\\d+) *sec *\\( *(\\d+[.]\\d+) *% *\\)")

func parseProgress(s string, prog *RTMPProgress) bool {
	matches := progressRegexp.FindStringSubmatch(s)
	if len(matches) != 4 {
		return false
	}
	for i, match := range matches[1:] {
		float, err := strconv.ParseFloat(match, 32)
		if err != nil {
			return false
		}
		if i == 0 {
			prog.KiloBytes = float32(float)
		} else if i == 1 {
			prog.Seconds = float32(float)
		} else {
			prog.Percent = float32(float)
		}
	}
	return true
}
