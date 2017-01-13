package rtmp

import "os/exec"

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
	t.cmd = exec.Command(ffmpegPath, "-re", "-i", t.path, "-f", "flv", t.url)
	t.cmd.Stdout = nil
	t.cmd.Stderr = nil
	return t.cmd.Run()
}
