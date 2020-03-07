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
    type: batch
    data:
      - weather,location=us-midwest temperature=80
      - weather,location=us-midwest temperature=82
    expects:
      ok: 0
      warn: 1
      crit: 0

```  

## Local development

Local golang code changes can be tested using the `replace`
feature of [Go modules](https://github.com/golang/go/wiki/Modules).

For instance, to test local changes to `io/influxdb.go`:

**Create go.mod inside the `./io` subdirectory**
```
 $ docker run -it \
     --mount type=bind,source="$(pwd)",target=/kapacitor-unit \
     --workdir /kapacitor-unit \
     golang:1.12.9-buster \
     /bin/bash

 /kapacitor-unit# cd io
 /kapacitor-unit/io# go mod init "kapacitor-unit/io"
 /kapacitor-unit/io# exit

 $ sudo chown $(stat Makefile --format='%u.%g') io/go.mod
```

**Add `replace` to go.mod**
```
require (
	...
	github.com/DreadPirateShawn/kapacitor-unit/io v0.0.0
)

replace github.com/DreadPirateShawn/kapacitor-unit/io => ./io
```

**Verify you succeeded**
This will run `go list -m all` in a Docker container, and should
reflect the above local replacement if you succeeded.
```
make go-list
```

Once you're finished testing, remeber to remove the subdirectory `go.mod`
and remove the require/replace pair added to go.mod.

**Note:** The above approach may change as Go versions increase, and
involves things like `chown` which aren't necessary if you're using locally
installed golang (rather than docker container isolation). It also reflects
the landscape that package management in Go can be an extremely dense topic --
this is at least one way to achieve the goal.

## Contributions

Fork and PR and use issues for bug reports, feature requests and general comments.

:copyright: MIT
