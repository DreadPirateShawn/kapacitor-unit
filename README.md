# Kapacitor-unit

**A test framework for TICKscripts**

[![Build Status](https://travis-ci.org/DreadPirateShawn/kapacitor-unit.svg?branch=master)](https://travis-ci.org/DreadPirateShawn/kapacitor-unit) ![Release Version](https://img.shields.io/badge/release-0.8-blue.svg)


Kapacitor-unit is a testing framework to make TICK scripts testing easy and
automated. Testing with Kapacitor-unit is as easy as defining the test configuration saying which alerts are expected to trigger when the TICK script processes specific data. 


Read more about the idea and motivation behind kapacitor-unit in 
[this blog post](http://www.gpestana.com/blog/kapacitor-unit/)


## Show me Kapacitor-unit in action!
![usage-example](https://media.giphy.com/media/xT0xetJEkloDtbVHSU/giphy.gif)


## Features

:heavy_check_mark: Run tests for **stream** TICK scripts using protocol line data input 

:heavy_check_mark: Run tests for **batch** TICK scripts using protocol line data input 

:soon: Run tests for **stream** and **batch** TICK scripts using recordings 


## Requirements

To run tests, both Kapacitor and Influx need to be running. (The latter is used for batch queries.)

These can be started using [docker-compose](https://docs.docker.com/compose/install/):
```
make start-kapacitor-and-influx
```

In order for all features to be supported, the Kapacitor version running the tests must be v1.3.4 or higher.

## Installing kapacitor-unit

**Binary from upstream:**
```
 $ curl -L https://github.com/DreadPirateShawn/kapacitor-unit/raw/master/main -o /usr/local/bin/kapacitor-unit
 $ chmod a+x /usr/local/bin/kapacitor-unit
```

**Building from source:**
```
 $ go install ./cmd/kapacitor-unit
 $ kapacitor-unit
```

**Running from source without rebuilding:**
```
 $ go run ./cmd/kapacitor-unit/main.go
```

Note that the Makefile uses a docker container to support testing / development
without locally installing golang.

## Running tests

```
kapacitor-unit --dir <*.tick directory> --kapacitor <kapacitor host> --influxdb <influxdb host> --tests <test configuration path>
```

### Test case definition:

```yaml

# Test case for alert_weather.tick
tests:
  
   # This is the configuration for a test case. The 'name' must be unique in the
   # same test configuration. 'description' is optional

  - name: Alert weather:: warning
    description: Task should trigger Warning when temperature raises about 80 

    # 'task_name' defines the name of the file of the tick script to be loaded
    # when running the test
    task_name: alert_weather.tick

    db: weather
    rp: default 
    type: stream

     # 'data' is an array of data in the line protocol
    data:
      - weather,location=us-midwest temperature=75
      - weather,location=us-midwest temperature=82

    # Alert that should be triggered by Kapacitor when test data is running 
    # against the task
    expects:
      ok: 0
      warn: 1
      crit: 0


  - name: Alert no. 2 using recording
    task_id: alert_weather.tick
    db: weather
    rp: default 
    type: stream
    recordind_id: 7c581a06-769d-45cb-97fe-a3c4d7ba061a
    expects:
      ok: 0
      warn: 1
      crit: 0


  - name: Alert no. 3 - Batch
    task_id: alert_weather.tick
    db: weather
    rp: default 
    duration: 30m
    type: batch
    data:
      - weather,location=us-midwest temperature=80 now()-1m
      - weather,location=us-midwest temperature=82 now()
    expects:
      ok: 0
      warn: 1
      crit: 0

```  

Note that `now() - 30m + 1m` relative influx timestamps can be used in batch data;
the only usage requirement is that `now()` needs to start the dynamic timestamp.
This currently only supports hours `h`, minute `m`, and seconds `s`.

### Debugging

Note that `log()` nodes added to tick script will appear in the kapacitor logs.
When running samples in this project, see [kapacitor config](infra/kapacitor/kapacitor.conf)
to determine whether that's a log file or STDERR, noting that the former (file)
will exist inside the target running kapacitor container.

## Contributions

Fork and PR and use issues for bug reports, feature requests and general comments.

:copyright: MIT
