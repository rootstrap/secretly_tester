#!/usr/bin/env python
import curses
from time import time, sleep
from csv import reader as parse_events
from sys import stdin
from collections import defaultdict
from itertools import islice

def infinite():
    i = 0
    while True:
        yield i
        i += 1

class Reporter(object):
    def __init__(self):
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
        self.instances = {}
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

class Session(object):
    def __init__(self, test_start_epoch):
        self.test_start_epoch = test_start_epoch
        self.streaming_start_epoch = None
        self.dropped = False
        self.bytes_sec_overall = 0
        self.bytes_sec_average = 0
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
    instance_metrics = ["KiloBytesSent", "KiloBytesRecv", "CPUUsage"]
    for e in events:
        session_id, stamp, metric, value = e
        stamp = float(stamp)
        if metric not in instance_metrics:
            if session_id not in metrics.sessions:
                metrics.sessions[session_id] = Session(stamp)
            if metric == 'StartTestOnMachine':
                metrics.sessions[session_id].instance_id = value
            if metric == 'ApiRequest':
                if metrics.sessions[session_id].instance_id == "":
                    metrics.sessions[session_id].instance_id = value
                metrics.requests += 1
                metrics.sessions[session_id].add_request(stamp)
            if metric == 'ApiRequestTimeout' or metric == 'ApiError':
                if metric == 'ApiRequestTimeout':
                    metrics.timeouts += 1
                if value == "critical":
                    del metrics.sessions[session_id]
            if metric == 'StreamProgressKiloBytes':
                metrics.sessions[session_id].update_kilobytes(stamp, float(value))
            if metric == 'StreamProgressSeconds':
                metrics.sessions[session_id].update_buffered(stamp, float(value))
                metrics.sessions[session_id].update_requests_average(stamp)
        else:
            if session_id not in metrics.instances:
                metrics.instances[session_id] = Instance()
            if metric == 'KiloBytesSent':
                metrics.instances[session_id].update_kilobytes_sent(stamp, float(value))
            if metric == 'KiloBytesRecv':
                metrics.instances[session_id].update_kilobytes_received(stamp, float(value))
            if metric == 'CPUUsage':
                metrics.instances[session_id].cpu_usage = float(value)
        on_update()

if __name__ == "__main__":
    metrics = Metrics()
    events = parse_events(iter(stdin.readline, ''))
    reporter = Reporter()
    try:
        fill_metrics(metrics, events, lambda: reporter.update(metrics))
    finally:
        reporter.stop()
