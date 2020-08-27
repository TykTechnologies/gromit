package util

// Send metrics to hostedgrapahite.com

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

var queue = make(chan string, 100)

func init() {
	go statsdSender()
}

func StatCount(metric string, value int) {
	queue <- fmt.Sprintf("%s.%s:%d|c", os.Getenv("STATS_API_KEY"), metric, value)
}

func StatTime(metric string, took time.Duration) {
	queue <- fmt.Sprintf("%s.%s:%d|ms", os.Getenv("STATS_API_KEY"), metric, took/1e6)
}

func StatGauge(metric string, value int) {
	queue <- fmt.Sprintf("%s.%s:%d|g", os.Getenv("STATS_API_KEY"), metric, value)
}

func statsdSender() {
	for s := range queue {
		if conn, err := net.Dial("udp", os.Getenv("STATS_SERVER")); err == nil {
			io.WriteString(conn, s)
			conn.Close()
		}
	}
}
