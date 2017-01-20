#!/usr/bin/env python
import curses
from time import time, sleep
from csv import reader as parse_events
from sys import stdin
from collections import defaultdict

class Reporter(object):
    def __init__(self):
        self.scr = curses.initscr()
        curses.noecho()
        curses.cbreak()

    def stop(self):
        curses.echo()
        curses.nocbreak()
        curses.endwin()

    def update(self, sessions):
        self.scr.addstr(0, 0, "%d sessions established" % len(sessions))
        streams_dropped = sum(1 for s in sessions.itervalues() if s.dropped)
        self.scr.addstr(2, 0, "Streams lagged: [{0:80}] {1}/{2}".format('#' * (streams_dropped / len(sessions) * 80), streams_dropped, len(sessions)))
        if sessions:
            avgrate = sum(s.bytes_sec_average for s in sessions.itervalues()) / len(sessions)
            self.scr.addstr(4, 0, "Average rate {0:8.2f} kbps".format(avgrate * 8))
        self.scr.addstr(6, 0, ("#" * 40) + " top 20")
        for i, (session_id, session) in enumerate(sessions.items()[:20]):
            self.scr.addstr(7 + i, 0, "Session {0}: overall rate {1:8.2f} kbps".format(session_id, session.bytes_sec_average * 8))
        self.scr.refresh()

class Session(object):
    def __init__(self, epoch):
        self.epoch = epoch
        self.dropped = False
        self.bytes_sec_overall = 0
        self.bytes_sec_average = 0
        self._last_kb = None

    def update_buffered(self, time, secs_buffered):
        self.dropped = time - self.epoch > secs_buffered + 3

    def update_kilobytes(self, time, kb):
        relative = time - self.epoch
        if relative > 0:
            self.bytes_sec_overall = kb / relative
        if self._last_kb:
            last_time, last_kb = self._last_kb
            if time - last_time > 1:
                self.bytes_sec_average = (kb - last_kb) / (time - last_time)
                self._last_kb = (time, kb)
        else:
            self._last_kb = (time, kb)


if __name__ == "__main__":
    sessions = {}

    reporter = Reporter()
    try:
        last = time()
        for e in parse_events(iter(stdin.readline, '')):
            session_id, stamp, metric, value = e
            stamp = float(stamp)
            if session_id not in sessions:
                sessions[session_id] = Session(stamp)
            if metric == 'StreamProgressKiloBytes':
                sessions[session_id].update_kilobytes(stamp, float(value))
            if metric == 'StreamProgressSeconds':
                sessions[session_id].update_buffered(stamp, float(value))
            if time() - last < 0.2:
                continue
            last = time()
            reporter.update(sessions)
    finally:
        reporter.stop()
