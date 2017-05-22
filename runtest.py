#!/usr/bin/env python
import curses
from time import time, sleep
from csv import reader as parse_events
from sys import stdin, stdout, stderr, exit
from collections import defaultdict
from itertools import islice
from subprocess import Popen, PIPE
from argparse import ArgumentParser, ArgumentTypeError
from select import select
from tempfile import TemporaryFile
from signal import SIGINT
import json
from threading import Thread


def main():
    def parse_duration(s):
        res = parse_duration_to_secs(s)
        if res is None:
            raise ArgumentTypeError("%s is not a valid duration (e.g. 1h30m12s5us)" % s)
        return res
    parser = ArgumentParser(description='Load test', add_help=False)
    parser.add_argument('--help', dest='help', action='store_true', default=False)
    parser.add_argument('--stdin', dest='stdin', action='store_true', default=False,
                                   help="don't invoke test, read test run on STDIN for debugging")
    parser.add_argument('--saveout', dest='saveout', action='store_true', default=False,
                                     help="save output in lastrun.stdout, lastrun.stderr")
    parser.add_argument('--ci', dest='ci', action='store_true', default=False,
                                help="run in CI mode, returning statistics")
    parser.add_argument('--steady-timeout', dest='steady_timeout', default=60, type=parse_duration,
                                help="seconds of steady state streaming to terminate test after (CI mode)")
    parser.add_argument('--timeout', dest='timeout', default=60*10, type=parse_duration,
                                help="seconds until hard terminating the test (CI mode)")
    args, unknown = parser.parse_known_args()

    if args.help:
        parser.print_help(stderr)
        print >> stderr
        TestInvokingDataSource.invoke(['-h']).wait()
        exit(2)

    if args.stdin:
        data_source = STDINDataSource(stdin)
    else:
        if args.saveout:
            data_source = TestInvokingDataSource(unknown, 'lastrun.stdout', 'lastrun.stderr')
        else:
            data_source = TestInvokingDataSource(unknown)

    metrics = Metrics()
    events = parse_events(data_source.get_lines())
    if args.ci:
        exiting = False
        def terminate():
            started = time()
            while time() - started < args.timeout:
                if exiting:
                    return
                sleep(1)
            print >> stderr, "timeout elapsed, terminating"
            data_source.stop()
            sleep(1)
            exit(1)
        Thread(target=terminate).start()
        try:
            def check_done():
                if metrics.all_streams_buffered(args.steady_timeout):
                    print >> stderr, "all streams buffered for %dsec" % args.steady_timeout
                    data_source.stop()
            fill_metrics(metrics, events, check_done)
        except KeyboardInterrupt:
            data_source.stop()
        else:
            json.dump(metrics.to_dict(), stdout)
            stdout.write('\n')
        finally:
            exiting = True
    else:
        reporter = Reporter()
        try:
            fill_metrics(metrics, events, lambda: reporter.update(metrics))
        except KeyboardInterrupt:
            data_source.stop()
        finally:
            reporter.stop()


class STDINDataSource(object):
    def __init__(self, stream):
        self.stream = stream
        self.stopped = False

    def get_lines(self):
        for line in iter(self.stream.readline, ''):
            yield line
            if self.stopped:
                return

    def stop(self):
        self.stopped = True


class TestInvokingDataSource(object):
    argv_prefixes = (
        ('talkative_stream_test',),
        ('go', 'run', 'main.go',),
    )

    def __init__(self, args, stdout_path=None, stderr_path=None):
        self.args = args
        self.stdout_path = stdout_path
        self.stderr_path = stderr_path
        self.popen = None
        self.stopped = False

    def get_lines(self):
        stdout_file = None
        if self.stdout_path:
            stdout_file = open(self.stdout_path, 'w+b')

        if self.stderr_path:
            stderr_file = open(self.stderr_path, 'w+b')
        else:
            stderr_file = TemporaryFile()

        self.popen = self.invoke(self.args, stdout=PIPE, stderr=stderr_file)

        while True:
            ready, _, _ = select((self.popen.stdout,), (), (), 0.2)
            stderr_file.seek(0, 1) # refresh
            data = stderr_file.read()
            if data:
                stderr.write(data)
            if self.popen.stdout in ready:
                break
            if self.stopped:
                return

        for line in iter(self.popen.stdout.readline, ''):
            yield line
            if stdout_file:
                stdout_file.write(line)
            if self.stopped:
                return

    def stop(self):
        self.stopped = True
        if self.popen and self.popen.returncode is None:
            self.popen.send_signal(SIGINT)
            sleep(3)
            if self.popen.poll() is None:
                self.popen.terminate()
            self.popen.wait()

    @classmethod
    def invoke(cls, args, **kwargs):
        argvs = [list(p) for p in cls.argv_prefixes]
        for argv in argvs:
            argv.extend(args)
            try:
                return Popen(argv, **kwargs)
            except OSError as e:
                if e.errno != 2:
                    raise
        else:
            print >> stderr, "Failed to invoke any of:"
            for argv in argvs:
                print >> stderr, ' '.join(argv)
            exit(1)


def infinite():
    i = 0
    while True:
        yield i
        i += 1


class Reporter(object):
    def __init__(self):
        self.inited = False

    def init(self):
        scr = curses.initscr()
        self.height, self.width = scr.getmaxyx()
        curses.endwin()
        self.left_window = curses.newwin(self.height, self.width / 2, 0, 0)
        self.divider_window = curses.newwin(self.height, 1, 0, self.width / 2 - 1)
        self.right_window = curses.newwin(self.height, self.width / 2, 0, self.width / 2)
        self.last_time_updated = time()
        curses.noecho()
        curses.cbreak()

    def stop(self):
        if self.inited:
            curses.echo()
            curses.nocbreak()
            curses.endwin()

    def update_left_window(self, metrics):
        self.left_window.clear()
        indexes = infinite()
        draw = lambda s: self.left_window.addstr(next(indexes), 0, s)

        draw(("#" * 40) + " API requests")
        draw("Overall average number of requests: {0:8.2f}/s".format(metrics.average_request_rate))
        draw("Total number of requests: {0}".format(metrics.requests))
        draw("Total number of timeouts: {0}".format(metrics.timeouts))
        draw(("#" * 40) + " API requests breakdown top 20")
        for instance_id, req_rate in islice(metrics.average_request_rate_by_instance, 0, 20):
            draw("Average number of requests for instance {0}: {1:8.2f}/s".format(instance_id, req_rate))

        next(indexes)
        draw(("#" * 40) + " Streaming")
        draw("%d/%d sessions established" % (metrics.num_sessions_established, metrics.num_sessions))
        streams_lagged_ratio = metrics.num_sessions_dropped / metrics.num_sessions if metrics.num_sessions else 0

        next(indexes)
        draw("Streams lagged: [{0:80}] {1}/{2}     ".format('#' * (streams_lagged_ratio * 80), metrics.num_sessions_dropped, metrics.num_sessions))
        if metrics.num_sessions:
            draw("Average rate {0:8.2f} kbps".format(metrics.average_byte_rate * 8))
        draw(("#" * 40) + " Streaming top 20")
        for session_id, byte_rate in islice(metrics.average_byte_rate_by_session, 0, 20):
            draw("Session {0}: overall rate {1:8.2f} kbps".format(session_id, byte_rate * 8))
        self.left_window.refresh()

    def update_right_window(self, metrics):
        self.right_window.clear()
        indexes = infinite()
        draw = lambda s: self.right_window.addstr(next(indexes), 0, s)

        draw(("#" * 40) + " Instances bitrates infos top 20")
        for instance_id, bit_rate_recv, bit_rate_sent in islice(metrics.average_bit_rate_by_instance, 0, 20):
            draw("Instance {0}: IN {1:12.2f} kbps | OUT {2:12.2f} kbps".format(instance_id, bit_rate_recv, bit_rate_sent))

        next(indexes)
        draw(("#" * 40) + " Instances CPU infos top 20")
        for instance_id, cpu_util in islice(metrics.average_cpu_by_instance, 0, 20):
            draw("Instance {0}: {1:3.2f}%".format(instance_id, cpu_util))
        self.right_window.refresh()

    def update_divider(self):
        for i in range(0, self.height - 1):
            self.divider_window.addstr(i, 0, "|")
        self.divider_window.refresh()

    def update(self, metrics):
        if not self.inited:
            self.init()
            self.inited = True
        now = time()
        if now - self.last_time_updated < 0.5:
            return
        self.update_left_window(metrics)
        self.update_divider()
        self.update_right_window(metrics)
        self.last_time_updated = now


class Metrics(object):
    def __init__(self):
        self.sessions = {}
        self.instances = defaultdict(Instance)
        self.requests = 0
        self.timeouts = 0

    @property
    def average_request_rate(self):
        return sum(s.avg_nb_requests for s in self.sessions.itervalues())

    @property
    def average_request_rate_by_instance(self):
        instances_request_avg = defaultdict(int)
        for session in self.sessions.itervalues():
            instances_request_avg[session.instance_id] += session.avg_nb_requests
        return instances_request_avg.iteritems()

    @property
    def num_sessions_established(self):
        return sum(1 for s in self.sessions.itervalues() if s.streaming_start_epoch is not None)

    @property
    def num_sessions(self):
        return len(self.sessions)

    @property
    def num_sessions_dropped(self):
        return sum(1 for s in self.sessions.itervalues() if s.dropped)

    @property
    def average_byte_rate(self):
        if len(self.sessions):
            return sum(s.bytes_sec_average for s in self.sessions.itervalues()) / len(self.sessions)

    @property
    def average_byte_rate_by_session(self):
        for session_id, session in self.sessions.iteritems():
            yield session_id, session.bytes_sec_average

    @property
    def average_bit_rate_by_instance(self):
        for instance_id, instance in self.instances.iteritems():
            yield instance_id, instance.bitrate_recv, instance.bitrate_sent

    @property
    def average_cpu_by_instance(self):
        for instance_id, instance in self.instances.iteritems():
            yield instance_id, instance.cpu_usage

    def all_streams_buffered(self, secs):
        if self.num_sessions == 0 or self.num_sessions > self.num_sessions_established:
            return False
        for session in self.sessions.itervalues():
            if session.secs_buffered < secs:
                return False
        return True

    def record_test_start(self, session_id, instance_id, stamp):
        self.sessions[session_id] = Session(stamp)
        self.sessions[session_id].instance_id = instance_id

    def record_api_request(self, session_id, instance_id, stamp):
        if self.sessions[session_id].instance_id == "":
            self.sessions[session_id].instance_id = instance_id
        self.requests += 1
        self.sessions[session_id].add_request(stamp)

    def record_api_request_error(self, session_id, stamp, level):
        if level == "critical":
            del self.sessions[session_id]

    def record_api_request_timeout(self, session_id, stamp):
        self.timeouts += 1

    def record_stream_progress_kilobytes(self, session_id, stamp, kilobytes):
        self.sessions[session_id].update_kilobytes(stamp, kilobytes)

    def record_stream_progress_seconds(self, session_id, stamp, seconds):
        self.sessions[session_id].update_buffered(stamp, seconds)
        self.sessions[session_id].update_requests_average(stamp)

    def record_instance_kilobytes_sent(self, instance_id, stamp, kb):
        self.instances[instance_id].update_kilobytes_sent(stamp, kb)

    def record_instance_kilobytes_received(self, instance_id, stamp, kb):
        self.instances[instance_id].update_kilobytes_received(stamp, kb)

    def record_instance_cpu_usage(self, instance_id, stamp, usage):
        self.instances[instance_id].cpu_usage = usage

    def to_dict(self):
        return {
            'sessions': self.num_sessions,
            'established': self.num_sessions_established
        }

class Session(object):
    def __init__(self, test_start_epoch):
        self.test_start_epoch = test_start_epoch
        self.streaming_start_epoch = None
        self.dropped = False
        self.bytes_sec_overall = 0
        self.bytes_sec_average = 0
        self.secs_buffered = 0
        self._last_kb = None

        self.instance_id = ""
        self.avg_nb_requests = 0
        self.nb_requests = 0
        self.nb_requests_timeout = 0

    def _set_streaming_start_epoch(self, time):
        if not self.streaming_start_epoch:
            self.streaming_start_epoch = time

    def update_buffered(self, time, secs_buffered):
        self._set_streaming_start_epoch(time)
        self.secs_buffered = secs_buffered
        self.dropped = time - self.streaming_start_epoch > secs_buffered + 3

    def update_kilobytes(self, time, kb):
        self._set_streaming_start_epoch(time)
        relative = time - self.streaming_start_epoch
        if relative > 0:
            self.bytes_sec_overall = kb / relative
        if self._last_kb:
            last_time, last_kb = self._last_kb
            if time - last_time > 1:
                self.bytes_sec_average = (kb - last_kb) / (time - last_time)
                self._last_kb = (time, kb)
        else:
            self._last_kb = (time, kb)

    def add_request(self, time):
        self.nb_requests += 1
        self.update_requests_average(time)

    def update_requests_average(self, time):
        if time != self.test_start_epoch:
            self.avg_nb_requests = self.nb_requests / (time - self.test_start_epoch)


class Instance(object):
    def __init__(self):
        self.cpu_usage = 0.0

        self.last_kb_recv = None
        self.bitrate_recv = 0

        self.last_kb_sent = None
        self.bitrate_sent = 0

    def update_kilobytes_received(self, time, kb):
        if self.last_kb_recv:
            last_time, last_kb = self.last_kb_recv
            if time - last_time > 1:
                self.bitrate_recv = (kb - last_kb) / (time - last_time)
                self.last_kb_recv = (time, kb)
        else:
            self.last_kb_recv = (time, kb)


    def update_kilobytes_sent(self, time, kb):
        if self.last_kb_sent:
            last_time, last_kb = self.last_kb_sent
            if time - last_time > 1:
                self.bitrate_sent = (kb - last_kb) / (time - last_time)
                self.last_kb_sent = (time, kb)
        else:
            self.last_kb_sent = (time, kb)


def fill_metrics(metrics, events, on_update):
    for e in events:
        entity_id, stamp, metric, value = e[:4]
        rest = e[4:]
        stamp = float(stamp)
        if metric == 'StartTestOnMachine':
            metrics.record_test_start(entity_id, value, stamp)
        elif metric == 'ApiRequest':
            metrics.record_api_request(entity_id, value, stamp)
        elif metric == 'ApiRequestTimeout':
            metrics.record_api_request_timeout(entity_id, stamp)
        elif metric == 'ApiError':
            metrics.record_api_request_error(entity_id, stamp, level)
        elif metric == 'StreamProgressKiloBytes':
            metrics.record_stream_progress_kilobytes(entity_id, stamp, float(value))
        elif metric == 'StreamProgressSeconds':
            metrics.record_stream_progress_seconds(entity_id, stamp, float(value))
        elif metric == 'KiloBytesSent':
            metrics.record_instance_kilobytes_sent(entity_id, stamp, float(value))
        elif metric == 'KiloBytesRecv':
            metrics.record_instance_kilobytes_received(entity_id, stamp, float(value))
        elif metric == 'CPUUsage':
            metrics.record_instance_cpu_usage(entity_id, stamp, float(value))
        on_update()

duration_units_to_ns = {
    'ns': 1,
    'us': 1000,
    'ms': 1000*1000,
    's': 1000*1000*1000,
    'm': 1000*1000*1000*60,
    'h': 1000*1000*1000*60*60
}

def parse_duration_to_secs(s):
    """
    mimick golang's Duration parsing

    >>> parse_duration_to_secs('1s')
    1.0
    >>> parse_duration_to_secs('1ns')
    1e-09
    >>> parse_duration_to_secs('1h3m12348us')
    3780.012348
    >>> parse_duration_to_secs('')
    >>> parse_duration_to_secs('1')
    >>> parse_duration_to_secs('1 ')
    >>> parse_duration_to_secs('1d')
    >>> parse_duration_to_secs('h1')
    """
    parts = []
    num_acc = ''
    unit_acc = ''
    for c in s:
        if '0' <= c <= '9':
            if not unit_acc:
                num_acc += c
            else:
                parts.append((int(num_acc), unit_acc))
                unit_acc = ''
                num_acc = c
        else:
            if not num_acc:
                return
            unit_acc += c
    if not num_acc or not unit_acc:
        return
    parts.append((int(num_acc), unit_acc))

    ns = 0
    for num, unit in parts:
        if unit not in duration_units_to_ns:
            return
        ns += duration_units_to_ns[unit] * num
    return float(ns) / 10**9

if __name__ == "__main__":
    main()
