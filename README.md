
# AQI2OTel: Air Quality to OpenTelemetry Adapter

I purchased a PurpleAir air quality sensor in the summer of 2022. While the
map and graphs on the [PurpleAir](https://purpleair.com) website are good,
what I really wanted was for something to _alert_ me when the air quality at
my home was getting bad. And as it turns out, I just happen know a service
that can take metric time series and generate alerts in various ways: I
decided to pipe my air quality data to [Lightstep](https://lightstep.com).

## Summary

This repo contains a small Go program that:

1. Queries the PurpleAir API for data associated with a single sensor.
2. Pushes those data using OpenTelemetry metrics to Lightstep.

It can also be used on the command line to test changes locally.

I run that code as a Google Cloud Function every two minutes. There a
step-by-step process for setting that up that below (mostly for the benefit of
myself in the future).

### Caveats

I'm sure many more that I've not listed here but to name a few:

- No error handling (unless you count `panic`)
- Possibly some not particularly idiomatic OTel things
- Not very consistent about configuration

# Code

There's not much to say about the code itself: it just does the two steps
listed above. I wrote it as a stand-alone program: it expects to be run
whenever it needs to scrape metrics from the PurpleAir and then exits
immediately after forwarding these metric values. (An alternative would have
been to have a long running process with an internal timer to wake up
periodically to perform measurements.)

Perhaps the only interesting thing is calling `Stop()` on the
`MeterProvider` - this forces the provider to flush any queued counters and to
make an observation of any asynchronous instruments. Don't forget to check the
result of `Stop()`!

Use standard Go commands to build, run, and test locally. Run `make zip` to
create a zip file suitable for uploading to Google Cloud.

## Go version

At the time I implemented this, the latest version of Go supported by Google
Cloud Functions was 1.16, so I included `go 1.16` in the go.mod file.

# Google Cloud Set-Up

## Secrets

Create [Secrets](https://console.cloud.google.com/security/secret-manager) to store...

- An OTel exporter header containing your Lightstep access token.
- Your Purple (read) API key

The access token itself needs to be URI encoded when passed in this way. It'll
look something like `lightstep-access-token=<access token possibly including
some %-escaped characters>`.

Don't forget to give the `Secret Manager Secret Accessor` role to the service
account that will run your function.

## Scheduling Your Function

### Cloud Pub/Sub

Create a [Pub/Sub
topic](https://console.cloud.google.com/cloudpubsub/topic/list) that will
connect the Cloud Schedule Job with your Cloud Function (both described
below). I left all of the defaults as is.

### Cloud Scheduler

Create a [Cloud Scheduler
Job](https://console.cloud.google.com/cloudscheduler). Given that the
PurpleAir sensor itself is only reporting values every 120 seconds, I used
`*/2 * * * *` as the cron schedule. Set the target type to "Pub/Sub" and
select the pub/sub topic you created above.

## Cloud Function

Create a new [function](https://console.cloud.google.com/functions) and give
it a name. Set the trigger type to "Cloud Pub/Sub" and select the topic you
created above. Under "Runtime, build, connections and security settings"
created settings as described below.

### Runtime environment variables

- `PURPLE_AIR_SENSOR_ID` - Set this to the sensor whose you'd like to
  scrape. Mine is a six digit number.

### Secrets

- `OTEL_EXPORTER_OTLP_HEADERS` - Set this to the secret you created above
  that contains the header with your Lightstep access token.

- `PURPLE_AIR_API_KEY` - Set this to the secret you created above that
  contains your PurpleAir API key.
  
### Code

Select Go 1.16 as the runtime. I found the "ZIP Upload" option to be
easiest. Use the zip file you created with make as described above. Set the
entry point to `Entry`.

# Local testing

It's a lot easier to test changes by running the code locally than it is to
upload it as a Cloud Function. To do so, first make a copy of the script that
sets environment variables.

    cp env.sh.example env.sh

Then set the variables in that file as you did for the function above.

If you just want to test scraping the PurpleAir API (or the light processing
that's performed on those results), you can use the stdout exporter as
follows:

    go run ./cmd --stdout

Without `--stdout` the program will send data to Lightstep.
