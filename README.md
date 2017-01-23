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

* Clone this repository into the src folder of your GO workspace (https://golang.org/doc/code.html#Overview)
* `$ export GOPATH=/path/to/your/go/workspace`
* Build the project:
    ```
    $ cd $GOPATH
    $ go get bitbucket.org/msolutionio/talkative_stream_test
    $ go install bitbucket.org/msolutionio/talkative_stream_test
    ```

### Build infrastructure ###

* Builds AMI using packer
    * Installs ffmpeg, rtmpdump, golang
    * Git exports latest or specified revision
    * Builds during AMI creation
    * Installs display script and others
* Builds infra using terraform, all nodes interchangeable

```
$ cd ${GOPATH}/src/bitbucket.org/msolutionio/talkative_stream_test/infra
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
$ ./build.sh
```

### Run test on built servers

```
$ terraform output
nodeip = [
    54.67.25.209,
    54.183.51.216,
    54.183.13.55
]
$ ssh -i files/id_rsa -l ubuntu 54.67.25.209 # any of the above IPs will work
$ talkative_runtest -help
  -email string
    	influencer email (default "hrant@msolution.io")
  -existingoffset int
    	sequence number offset for existing users
  -percentnew int
    	0-100 percentage of new fan users in test
  -ramp duration
    	time between users joining, e.g. 200ms (default 500ms)
  -sshhosts string
    	space separated list of user@host to run test on (clustered)
  -sshkeyfile string
    	path to SSH private key file
  -token string
    	influencer token
  -users int
    	number of concurrent users (default 10)
$ talkative_runtest --users 10 # runs test with 10 users
```
