# CS425-MP2

## Usage

	$ docker build -t cs425-mp2 .
	$ docker run -it --rm --name cs425-mp2 cs425-mp2

## Usage without docker

	$ go run main.go

The default output is /tmp/mp2.log, use:

	$ tail -f /tmp/mp2.log

to monitor the log.
