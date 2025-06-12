## Liveness Observer for Services

### Usage:

    // probe will be failed after 15 seconds
    probe := liveness.NewProbe(time.Second*15)

    service1 := usefulService1.New()
    service2 := usefulService2.New()

    probe.Watch(service1, service2)

    if !probe.IsAlive() {
        panic("observable services are down")
    }


    