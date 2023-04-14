## GCP Spot Check
### Background
TODO
### Problem
GCP Compute Engine Spot instances do not have an automatic restart option, so this Go program does just that. Get a credentials file for a service account, fill out the environment file, build the container, then set up a cron job to automatically restart your instance when it goes down. Emails will be sent to notify you of the new IP address.

### Setup
1. Use the `sample.env` file to create a `.env` file
2. Create a new json key file for a GCP service account which has access to Compute Engine
3. Both of those files should be put in the `/gcp-spot-check` directory
4. Build the container from by running `make build` from the project root
5. Set up a cron job to run the container on an interval, mine looks like this:

```
*/2 * * * * docker run -t homelab-utils/gcp-spot-check 2>&1 | logger -t gcp-spot-check
```

The `*/2` indicates that the job should run every 2 minutes, you can change this to whatever polling interval you'd like

6. Enjoy not having to manually restart your VM every time it gets preempted.

### Alternative Solutions
- I've explored other solutions, creating a polling application in Go (running in a loop instead of cron schedule), using a VM shutdown script to call a cloud function that will trigger the restart, but this has been the best solution so far
- Of course the obvious is just to use a standard provisioning model and not a spot instance, but that'll cost you a couple extra bucks a month.