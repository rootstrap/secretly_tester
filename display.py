#!/usr/bin/env python
import curses
from time import time, sleep
from csv import reader as parse_events
from sys import stdin
from collections import defaultdict

class Reporter(object):
    def __init__(self):
        scr = curses.initscr()
        self.height, self.width = scr.getmaxyx()
        curses.endwin()
        self.left_window = curses.newwin(self.height, self.width / 2, 0, 0)
        self.divider_window = curses.newwin(self.height, 1, 0, self.width / 2 - 1)
        self.right_window = curses.newwin(self.height, self.width / 2, 0, self.width / 2)
        self.current_display_index = None
        self.last_time_updated_right_screen = time()
        curses.noecho()
        curses.cbreak()

    def stop(self):
        curses.echo()
        curses.nocbreak()
        curses.endwin()

    def get_display_index(self, offset=0, left=True):
        def _get_window_index():
            if left:
                return self.current_display_index['left']
            else:
                return self.current_display_index['right']
        if self.current_display_index == None:
            self.current_display_index = {'left': 0, 'right': 0}
            return _get_window_index()
        if left:
            self.current_display_index['left'] += offset + 1
        else:
            self.current_display_index['right'] += offset + 1
        return _get_window_index()

    def update_left_window(self, sessions, instances, requests, timeouts):
        self.left_window.clear()
        self.left_window.addstr(self.get_display_index(), 0, ("#" * 40) + " API requests")
        self.left_window.addstr(self.get_display_index(), 0, "Overall average number of requests: {0:8.2f}/s".format(sum(s.avg_nb_requests for s in sessions.itervalues())))
        self.left_window.addstr(self.get_display_index(), 0, "Total number of requests: {0}".format(requests))
        self.left_window.addstr(self.get_display_index(), 0, "Total number of timeouts: {0}".format(timeouts))
        instances_request_avg = defaultdict(list)
        for session in sessions.itervalues():
            instances_request_avg[session.instance_id].append(session.avg_nb_requests)
        self.left_window.addstr(self.get_display_index(), 0, ("#" * 40) + " API requests breakdown top 20")
        for i, (instance_id, values) in enumerate(instances_request_avg.items()[:20]):
            self.left_window.addstr(self.get_display_index(), 0, "Average number of requests for instance {0}: {1:8.2f}/s".format(instance_id, sum(values)))

        self.left_window.addstr(self.get_display_index(1), 0, ("#" * 40) + " Streaming")
        self.left_window.addstr(self.get_display_index(), 0, "%d/%d sessions established" % (sum(1 for s in sessions.itervalues() if s.streaming_start_epoch is not None), len(sessions)))
        streams_dropped = sum(1 for s in sessions.itervalues() if s.dropped)
        streams_lagged_ratio = streams_dropped / len(sessions) if len(sessions) else 0
        self.left_window.addstr(self.get_display_index(1), 0, "Streams lagged: [{0:80}] {1}/{2}     ".format('#' * (streams_lagged_ratio * 80), streams_dropped, len(sessions)))
        if sessions:
            avgrate = sum(s.bytes_sec_average for s in sessions.itervalues()) / len(sessions)
            self.left_window.addstr(self.get_display_index(), 0, "Average rate {0:8.2f} kbps".format(avgrate * 8))
        self.left_window.addstr(self.get_display_index(), 0, ("#" * 40) + " Streaming top 20")
        for i, (session_id, session) in enumerate(sessions.items()[:20]):
            self.left_window.addstr(self.get_display_index(), 0, "Session {0}: overall rate {1:8.2f} kbps".format(session_id, session.bytes_sec_average * 8))
        self.left_window.refresh()

    def update_right_window(self, sessions, instances, requests, timeouts):
        self.right_window.clear()
        self.right_window.addstr(self.get_display_index(0, False), 0, ("#" * 40) + " Instances bitrates infos top 20")
        for i, (instance_id, instance) in enumerate(instances.items()[:20]):
            self.right_window.addstr(self.get_display_index(0, False), 0, "Instance {0}: IN {1:12.2f} kbps | OUT {2:12.2f} kbps".format(instance_id, instance.bitrate_recv, instance.bitrate_sent))
        self.right_window.addstr(self.get_display_index(1, False), 0, ("#" * 40) + " Instances CPU infos top 20")
        for i, (instance_id, instance) in enumerate(instances.items()[:20]):
            self.right_window.addstr(self.get_display_index(0, False), 0, "Instance {0}: {1:3.2f}%".format(instance_id, instance.cpu_usage))
        self.right_window.refresh()

    def update_divider(self):
        for i in range(0, self.height - 1):
            self.divider_window.addstr(i, 0, "|")
        self.divider_window.refresh()

    def update(self, sessions, instances, requests, timeouts):
        self.current_display_index = None
        self.update_left_window(sessions, instances, requests, timeouts)
        if time() - last < 2.0:
            self.update_divider()
            self.update_right_window(sessions, instances, requests, timeouts)
        self.last_time_updated_right_screen = time()


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

instance_metrics = ["KiloBytesSent", "KiloBytesRecv", "CPUUsage"]

if __name__ == "__main__":
    sessions = {}
    instances = {}
    requests = 0
    timeouts = 0

    reporter = Reporter()
    try:
        last = time()
        for e in parse_events(iter(stdin.readline, '')):
            session_id, stamp, metric, value = e
            stamp = float(stamp)
            if metric not in instance_metrics:
                if session_id not in sessions:
                    sessions[session_id] = Session(stamp)
                if metric == 'StartTestOnMachine':
                    sessions[session_id].instance_id = value
                if metric == 'ApiRequest':
                    if sessions[session_id].instance_id == "":
                        sessions[session_id].instance_id = value
                    requests += 1
                    sessions[session_id].add_request(stamp)
                if metric == 'ApiRequestTimeout' or metric == 'ApiError':
                    if metric == 'ApiRequestTimeout':
                        timeouts += 1
                    if value == "critical":
                        del sessions[session_id]
                if metric == 'StreamProgressKiloBytes':
                    sessions[session_id].update_kilobytes(stamp, float(value))
                if metric == 'StreamProgressSeconds':
                    sessions[session_id].update_buffered(stamp, float(value))
                    sessions[session_id].update_requests_average(stamp)
            else:
                if session_id not in instances:
                    instances[session_id] = Instance()
                if metric == 'KiloBytesSent':
                    instances[session_id].update_kilobytes_sent(stamp, float(value))
                if metric == 'KiloBytesRecv':
                    instances[session_id].update_kilobytes_received(stamp, float(value))
                if metric == 'CPUUsage':
                    instances[session_id].cpu_usage = float(value)
            if time() - last < 0.5:
                continue
            last = time()
            reporter.update(sessions, instances, requests, timeouts)
    finally:
        reporter.stop()
