package io

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"github.com/influxdata/influxdb1-client/models"
	"github.com/PaesslerAG/gval"
	"io/ioutil"
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
	// 'now()' search -- case-insensitive to match influx, must follow whitespace.
	// most reliable (and cleanest) is just to capture all line parts,
	// rather than using 'now()' index to disassemble.
	lineRegex = regexp.MustCompile(`(.*)\s+[Nn][Oo][Ww]\(\)(.*)`)
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
	lineParts := lineRegex.FindStringSubmatch(value)
	if lineParts == nil {
		// if not found, use timestamp as-is
		return value, nil
	}
	timeSegment := now+lineParts[2]
	// replace all human-readable '1m', '30s' with corresponding nanoseconds
	newTimeSegment := string(regexpTimeDuration.ReplaceAllFunc([]byte(timeSegment), humanToNanoTime))
	// evaluate math formula, e.g. '5-4' returns '1'
	timeMathResult, err := evaluateMath(newTimeSegment)
	if err != nil {
		glog.Errorf("Processing error: value '%s' error: %s", value, err)
		return value, err
	}
	// concat original string up to the 'now()' + final timestamp verdict
	return lineParts[1]+" "+timeMathResult, nil
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

// Monitor for db create/delete
func (influxdb Influxdb) MonitorCreate(db string) error {
	glog.Info("DEBUG:: Influxdb create monitor ", db)
	attempts := 10
	for attempts > 0 {
		verdict, err := influxdb.DoesDatabaseExist(db)
		if err != nil {
			return err
		}
		if verdict {
			return nil
		}
		attempts--
		time.Sleep(time.Second)
	}
	return errors.New("Database not found: "+db)
}
func (influxdb Influxdb) MonitorDelete(db string) error {
	glog.Info("DEBUG:: Influxdb delete monitor ", db)
	attempts := 10
	for attempts > 0 {
		verdict, err := influxdb.DoesDatabaseExist(db)
		if err != nil {
			return err
		}
		if !verdict {
			return nil
		}
		attempts--
		time.Sleep(time.Second)
	}
	return errors.New("Database still found: "+db)
}

// Reference:
// https://github.com/influxdata/influxdb1-client/blob/master/influxdb.go
// (Loosely translated, no error handling, etc.)
type Result struct {
	Series   []models.Row
}
type Response struct {
	Results []Result
}

func (influxdb Influxdb) DoesDatabaseExist(db string) (bool, error) {
	glog.Info("DEBUG:: Influxdb checking database ", db)
	q := "q=SHOW DATABASES"
	baseUrl := influxdb.Host + "/query"
	resp, err := influxdb.Client.Post(baseUrl, "application/x-www-form-urlencoded",
		bytes.NewBuffer([]byte(q)))
	if err != nil {
		return false, err
	}
	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		return false, err2
	}
	var listResponse Response
	json.Unmarshal(body, &listResponse)
	for _, result := range listResponse.Results {
		for _, series := range result.Series {
			for _, value := range series.Values {
				for _, subvalue := range value {
					if subvalue == db {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// Creates db and rp where tests will run
func (influxdb Influxdb) Setup(db string, duration string, rp string) error {
	glog.Info("DEBUG:: Influxdb setup ", db+":"+rp)
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
	if err2 := influxdb.MonitorCreate(db); err2 != nil {
		return err2
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
	if err2 := influxdb.MonitorDelete(db); err2 != nil {
		return err2
	}
	glog.Info("DEBUG:: Influxdb cleanup database ", q)
	return nil
}
