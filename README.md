# README #

Talkative streaming test

### High level design
* Golang test
    * Creates influencer
    * Pre-creates (some) fans, follows influencer
    * Starts stream
    * Starts spinning up fans that watch influencer's stream (either in process or over SSH remotely)
    * Collects CSV log lines that show stream progress
* Python visualizer
    * Reads CSV log lines and outputs a basic UI with bitrates and N stream lagged

### Setup dev environment ###

* `$ export GOPATH=/path/to/your/go/workspace`
* Clone this repository into the src folder of your GO workspace `git clone ... $GOPATH/src/toptier/secretly_tester` (https://golang.org/doc/code.html#Overview)
* Build the project:
    ```
    $ cd $GOPATH/src/github.com/toptier/secretly_tester
    $ go install
    ```
* Add dependencies to `vendor`
    ```
    $ govendor get github.com/toptier/secretly_tester
    ```

### Build infrastructure ###

* Builds AMI using packer
    * Installs ffmpeg, rtmpdump, golang
    * Git exports latest or specified revision
    * Builds during AMI creation
    * Installs display script and others
* Builds infra using terraform, all nodes interchangeable

#### Install prerequisites

On mac OS X with homebrew
```
$ brew install terraform
$ brew install packer
$ brew install git
```

otherwise download and install from

https://www.terraform.io/intro/getting-started/install.html
https://www.packer.io/intro/getting-started/setup.html

and use your system's package management to install git

#### Creating infrastructure

```
$ cd ${GOPATH}/src/github.com/toptier/secretly_tester/infra # or wherever you've unpacked this repo
$ cat config.json # adjust config.json for right number of instances etc.
{
    "aws_access_key": "AKIAFEQWEQ",
    "aws_secret_key": "4321ljwrqlkjrwq+eqwegqljrqoijfaslkjeqw",
    "aws_region": "us-west-1",
    "instance_type": "m3.medium",
    "number_instances": "3",
    "security_group_id": "sg-d2e0ecb6",
    "subnet_id": "subnet-f0b881a9"
}
$ ./loadtest.sh build
```

### Run test on built servers

```
$ ./loadtest.sh ssh
$ talkative_runtest --help
Usage of talkative_stream_test runtest:
  -email string
        influencer email (default "hrant@msolution.io")
  -existingoffset int
        sequence number offset for existing users
  -percentnew int
        0-100 percentage of new fan users in test
  -precreatefans
        pre-create fans and follow influencer (not needed on repeat runs)
  -ramp duration
        time between users joining, e.g. 200ms (default 500ms)
  -sleepbetweensteps
  	Sleep between steps as a fan
  -sshhosts string
        space separated list of user@host to run test on (clustered)
  -sshkeyfile string
        path to SSH private key file
  -token string
        influencer token (default "4352915049.1677ed0.13fb746250c84b928b37360fba9e4d57")
  -timeout duration
    	Response time before timeout, e.g. 500ms
  -users int
        number of concurrent users (default 10)
  -videopath string
        path to video file used in test (default "640.flv")
/usr/local/bin/talkative_runtest: line 26:  3620 Terminated              tail -n +0 -f lastrun.stderr 1>&2
$ talkative_runtest --users 10 # runs test with 10 users
```

### Teardown infrastructure

```
$ cd ${GOPATH}/src/github.com/toptier/secretly_tester/infra # or wherever you've unpacked this repo
$ ./loadtest.sh destroy
```
