package rtmp

import (
	"bufio"
	"io"
	"os/exec"
	"regexp"
	"strconv"
)

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
	t.cmd.Start()
	sendProgresses(stream, t.Progress)
	return t.cmd.Wait()
}

type RTMPProgress struct {
	Seconds, KiloBytes float32
}

func sendProgresses(stream io.Reader, sink chan RTMPProgress) error {
	buff := bufio.NewReader(stream)
	var acc []byte
	var b byte
	var err error = nil
	for err == nil {
		b, err = buff.ReadByte()
		switch b {
		case ')':
			acc = append(acc, b)
			var progress RTMPProgress
			if parseProgress(string(acc), &progress) {
				sink <- progress
			}
		case 'c':
			acc = append(acc, b)
			var progress RTMPProgress
			if parseProgress(string(acc), &progress) {
				sink <- progress
			}
		case '\n', '\r':
			acc = acc[:0]
		default:
			acc = append(acc, b)
		}
	}
	if err != io.EOF {
		return err
	}
	return nil
}

var progressRegexp *regexp.Regexp = regexp.MustCompile("^ *(\\d+[.]\\d+) *kB +/ +(\\d+[.]\\d+) *sec *$")
var progressRegexpPercent *regexp.Regexp = regexp.MustCompile("^ *(\\d+[.]\\d+) *kB +/ +(\\d+[.]\\d+) *sec *\\( *(\\d+[.]\\d+) *% *\\) *$")

func parseProgress(s string, prog *RTMPProgress) bool {
	matches := progressRegexpPercent.FindStringSubmatch(s)
	if len(matches) != 4 {
		matches = progressRegexp.FindStringSubmatch(s)
		if len(matches) != 3 {
			return false
		}
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
		}
	}
	return true
}
