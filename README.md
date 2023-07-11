# Scootin Aboot

## Getting started

Once you want to test how the app is working main is set up the way you can run multiple clients that use the app.
To make it easier just use:

```
    docker compose up 
```

this should run the app and setup basic data in Redis. The logs are printed to stdout, so in the container
you would be able to see the rental processes in action.


## Architecture

I have chosen the modular monolith architecture as it is easier to build from scratch, easier to maintain
at the beginning of the process of creating app's prototype and app itself. If needed it is easy to break 
apart the modules and create microservices out of it enabling better scaling and adding more complexity.
The architecture's scratch can be seen in ArchitectuteScratch.drawio, it needs to be opened on draw.io website.

For the modules and resources seen on the architecture graph I have chosen:
- Redis for database, as it has basic functionalities for geospatial indexing is fast to read and easy to maintain. The
tradeoff here is that the geospatial functions even though present are not widely developed and that Redis was not meant
to handle multiple simultaneous operations using geospatial indexes, but is enough for a prototype app. In case of
further development it would be better to switch to sth more appropriate, like RedisGears.
- BFF service being just the service itself, shown on diagram as separate module to increase readability.

