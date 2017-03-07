package rtmp

import "path"
import "os/exec"
import "io/ioutil"
import "os"

type rtmpPush struct {
	url  string
	path string
	cmd  *exec.Cmd
}

func NewRTMPPusher(rtmpUrl string, path string) *rtmpPush {
	push := new(rtmpPush)
	push.url = rtmpUrl
	push.path = path
	return push
}

func (t *rtmpPush) Run() error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return err
	}
	file, err := ioutil.TempFile("", "talkativetestconcatenation")
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	for i := 0; i < nLoops; i++ {
		_, err := file.WriteString("file '" + path.Join(dir, t.path) + "'\n")
		if err != nil {
			file.Close()
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	t.cmd = exec.Command(ffmpegPath, "-f", "concat", "-safe", "0", "-re", "-i", file.Name(), "-acodec", "copy", "-vcodec", "copy", "-f", "flv", t.url)
	t.cmd.Stdout = nil
	t.cmd.Stderr = nil
	return t.cmd.Run()
}

var nLoops = 200
