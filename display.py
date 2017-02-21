#!/usr/bin/env python
import curses
from time import time, sleep
from csv import reader as parse_events
from sys import stdin
from collections import defaultdict

class Reporter(object):
    def __init__(self):
        self.scr = curses.initscr()
        self.current_display_index = None
        curses.noecho()
        curses.cbreak()

    def stop(self):
        curses.echo()
        curses.nocbreak()
        curses.endwin()

    def get_display_index(self, offset=0):
        if self.current_display_index == None:
            self.current_display_index = 0
            return self.current_display_index
        self.current_display_index += offset + 1
        return self.current_display_index

    def update(self, sessions, requests, timeouts):
        self.current_display_index = None
        self.scr.clear()
        self.scr.addstr(self.get_display_index(), 0, ("#" * 40) + " API")
        self.scr.addstr(self.get_display_index(), 0, "Overall average number of requests: {0:8.2f}/s".format(sum(s.avg_nb_requests for s in sessions.itervalues())))
        self.scr.addstr(self.get_display_index(), 0, "Total number of requests: {0}".format(requests))
        self.scr.addstr(self.get_display_index(), 0, "Total number of timeouts: {0}".format(timeouts))
        instances_request_avg = defaultdict(list)
        for session in sessions.itervalues():
            instances_request_avg[session.instance_id].append(session.avg_nb_requests)
        self.scr.addstr(self.get_display_index(), 0, ("#" * 40) + " API breakdown")
        for instance_id, values in instances_request_avg.iteritems():
            self.scr.addstr(self.get_display_index(), 0, "Average number of requests for instance {0}: {1:8.2f}/s".format(instance_id, sum(values)))

        self.scr.addstr(self.get_display_index(1), 0, ("#" * 40) + " Streaming")
        self.scr.addstr(self.get_display_index(), 0, "%d sessions established" % len(sessions))
        streams_dropped = sum(1 for s in sessions.itervalues() if s.dropped)
        self.scr.addstr(self.get_display_index(1), 0, "Streams lagged: [{0:80}] {1}/{2}     ".format('#' * (streams_dropped / len(sessions) * 80), streams_dropped, len(sessions)))
        if sessions:
            avgrate = sum(s.bytes_sec_average for s in sessions.itervalues()) / len(sessions)
            self.scr.addstr(self.get_display_index(), 0, "Average rate {0:8.2f} kbps".format(avgrate * 8))
        self.scr.addstr(self.get_display_index(), 0, ("#" * 40) + " Streaming top 20")
        for i, (session_id, session) in enumerate(sessions.items()[:20]):
            self.scr.addstr(self.get_display_index(), 0, "Session {0}: overall rate {1:8.2f} kbps".format(session_id, session.bytes_sec_average * 8))
        self.scr.refresh()

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

if __name__ == "__main__":
    sessions = {}
    requests = 0
    timeouts = 0

    reporter = Reporter()
    try:
        last = time()
        for e in parse_events(iter(stdin.readline, '')):
            session_id, stamp, metric, value = e
            stamp = float(stamp)
            if session_id not in sessions:
                sessions[session_id] = Session(stamp)
            if metric == 'StartTestOnMachine':
                sessions[session_id].instance_id = value
            if metric == 'ApiRequest':
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
            if time() - last < 0.2:
                continue
            last = time()
            reporter.update(sessions, requests, timeouts)
    finally:
        reporter.stop()
