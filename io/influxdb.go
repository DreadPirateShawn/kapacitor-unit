package io

import (
	"bytes"
	"github.com/golang/glog"
	"github.com/PaesslerAG/gval"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// Influxdb service configurations
type Influxdb struct {
	Host   string
	Client http.Client
}

func NewInfluxdb(host string) Influxdb {
	return Influxdb{
		host,
		http.Client{},
	}
}

var (
	// Matches '30s', '1h30m', etc -- case-sensitive to match influx
	// Only supports hours (h), minutes (m), seconds (s).
	regexpTimeDuration = regexp.MustCompile("([0-9]+[smh])+")
	// 'now()' search -- case-insensitive to match influx
	nowStr = "now()"
	regexpNow = regexp.MustCompile("(?i)"+nowStr)
	now = strconv.FormatInt(time.Now().UnixNano(), 10)
)

// Convert (e.g.) '5m' to '5 * nanoseconds per minute'
func humanToNanoTime(value []byte) ([]byte) {
	dura, err := time.ParseDuration(string(value))
	if err != nil {
	    return value
	}
	return []byte(strconv.FormatInt(dura.Nanoseconds(), 10))
}

// Perform math based on string input, e.g.
// input: 1257894000000000000-(3600000000000-60000000000)
// output: 1257890460000000000
func evaluateMath(value string) (string, error) {
	timeMathResult, err := gval.Evaluate(value, map[string]interface{}{})
	if err != nil {
	    return value, err
	}
	return strconv.FormatFloat(timeMathResult.(float64), 'f', 0, 64), nil
}

// return line with time segment replaced by dynamically-calculated timestamp
func translateTimestamp(value string) (string, error) {
	// find 'now()'
	posNowResp := regexpNow.FindStringIndex(value)
	if posNowResp == nil {
		// if not found, use timestamp as-is
		return value, nil
	}
	posNow := posNowResp[0]
	posNowEnd := posNow + len(nowStr)
	// if found, treat from 'now()' until end of line as time formula
	timeSegment := now+value[posNowEnd:len(value)]
	// replace all human-readable '1m', '30s' with corresponding nanoseconds
	newTimeSegment := string(regexpTimeDuration.ReplaceAllFunc([]byte(timeSegment), humanToNanoTime))
	// evaluate math formula, e.g. '5-4' returns '1'
	timeMathResult, err := evaluateMath(newTimeSegment)
	if err != nil {
	    return value, err
	}
	// concat original string up to the 'now()' + final timestamp verdict
	return value[0:posNow]+timeMathResult, nil
}

// Adds test data to influxdb
func (influxdb Influxdb) Data(data []string, db string, rp string) error {
	url := influxdb.Host + influxdb_write + "db=" + db + "&rp=" + rp
	for _, d := range data {
		d2, errTranslate := translateTimestamp(d)
		if errTranslate == nil {
			d = d2
		}
		_, err := influxdb.Client.Post(url, "application/x-www-form-urlencoded",
			bytes.NewBuffer([]byte(d)))
		if err != nil {
			return err
		}
		glog.Info("DEBUG:: Influxdb added ["+d+"] to "+url)
	}
	return nil
}

// Creates db and rp where tests will run
func (influxdb Influxdb) Setup(db string, duration string, rp string) error {
	glog.Info("DEGUB:: Influxdb setup ", db+":"+rp)
	// If no retention policy is defined, use "autogen"
	if rp == "" {
		rp = "autogen"
	}
	// If no duration is defined, use "1h" (original hardcoded value, to reduce regression risk)
	if duration == "" {
		duration = "1h"
	}
	q := "q=CREATE DATABASE \""+db+"\" WITH DURATION "+duration+" REPLICATION 1 NAME \""+rp+"\""
	baseUrl := influxdb.Host + "/query"
	_, err := influxdb.Client.Post(baseUrl, "application/x-www-form-urlencoded",
		bytes.NewBuffer([]byte(q)))
	if err != nil {
		return err
	}
	return nil
}

func (influxdb Influxdb) CleanUp(db string) error {
	q := "q=DROP DATABASE \""+db+"\""
	baseUrl := influxdb.Host + "/query"
	_, err := influxdb.Client.Post(baseUrl, "application/x-www-form-urlencoded",
		bytes.NewBuffer([]byte(q)))
	if err != nil {
		return err
	}
	glog.Info("DEBUG:: Influxdb cleanup database ", q)
	return nil
}
