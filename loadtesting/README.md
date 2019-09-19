# Load testing

You may wonder how Gotenberg behaves under load.

In order to help you having an idea, we created a bunch of scenarios for the
[k6](https://docs.k6.io/docs) load testing tool.

The Gotenberg container (version `6.0.0` and default options) was hosted on a AWS EC2 `t2.micro` instance (1 vCPU and 1 Go of RAM, low to average network performances).

The k6 scenarios have been performed on a MacBook Pro 2016 (2 GHz Intel Core i5 and 16 Go 1867 MHz LPDDR3).

## HTML

The HTML scenario is quite simple:

* Ramp up from 0 to 100 virtual users, each one uploading as many time as possible the [HTML test data](../test/testdata/html).
* Stop when at least one response is not an HTTP 200 code.

```bash
$ k6 run --env MAX_VUS=100 --env BASE_URL=http://ec2-foo.eu-west-1.compute.amazonaws.com html.js

          /\      |‾‾|  /‾‾/  /‾/   
     /\  /  \     |  |_/  /  / /    
    /  \/    \    |      |  /  ‾‾\  
   /          \   |  |‾\  \ | (_) | 
  / __________ \  |__|  \__\ \___/ .io

  execution: local
     output: -
     script: html.js

    duration: -,  iterations: -
         vus: 1, max: 100

    done [==========================================================] 1m11.9s / 10m0s

    ✗ is status 200
     ↳  99% — ✓ 179 / ✗ 1
    ✗ is not status 504
     ↳  99% — ✓ 179 / ✗ 1
    ✓ is not status 500

    checks.....................: 99.62% ✓ 538   ✗ 2    
    data_received..............: 41 MB  576 kB/s
    data_sent..................: 8.5 MB 118 kB/s
  ✗ failed requests............: 1      0.013895/s
    http_req_blocked...........: avg=1.58ms   min=2µs   med=5µs    max=59.73ms  p(90)=10.19µs  p(95)=20.57ms 
    http_req_connecting........: avg=1.56ms   min=0s    med=0s     max=59.61ms  p(90)=0s       p(95)=20.48ms 
    http_req_duration..........: avg=2.3s     min=1.32s med=1.67s  max=10.2s    p(90)=4.41s    p(95)=7.54s   
    http_req_receiving.........: avg=99.01ms  min=127µs med=98.4ms max=154.86ms p(90)=115.92ms p(95)=122.42ms
    http_req_sending...........: avg=232.69µs min=133µs med=207µs  max=1.16ms   p(90)=335.4µs  p(95)=382.64µs
    http_req_tls_handshaking...: avg=0s       min=0s    med=0s     max=0s       p(90)=0s       p(95)=0s      
    http_req_waiting...........: avg=2.2s     min=1.21s med=1.56s  max=10.2s    p(90)=4.32s    p(95)=7.45s   
    http_reqs..................: 180    2.501138/s
    iteration_duration.........: avg=2.3s     min=1.33s med=1.67s  max=10.22s   p(90)=4.45s    p(95)=7.54s   
    iterations.................: 180    2.501138/s
    vus........................: 12     min=1   max=12 
    vus_max....................: 100    min=100 max=100
```

In our use case, when reaching 12 virtual users (~2,5 requests per second), some incoming requests cannot be fulfilled before 10 seconds (`DEFAULT_WAIT_TIMEOUT` value).
During this test, CPU usage went from 0 to 100% and memory usage stayed low (Google Chrome go from 64.1 MB to 64.9 MB).

## Office

The Office scenario is the same as the HTML scenario, but with a [document.docx](../test/testdata/office/document.docx).

```bash
$ k6 run --env MAX_VUS=100 --env BASE_URL=http://ec2-foo.eu-west-1.compute.amazonaws.com office.js

          /\      |‾‾|  /‾‾/  /‾/   
     /\  /  \     |  |_/  /  / /    
    /  \/    \    |      |  /  ‾‾\  
   /          \   |  |‾\  \ | (_) | 
  / __________ \  |__|  \__\ \___/ .io

  execution: local
     output: -
     script: office.js

    duration: -,  iterations: -
         vus: 1, max: 100

    done [==========================================================] 2m7.9s / 10m0s

    ✗ is status 200
     ↳  99% — ✓ 481 / ✗ 3
    ✗ is not status 504
     ↳  99% — ✓ 481 / ✗ 3
    ✓ is not status 500

    checks.....................: 99.58% ✓ 1446  ✗ 6    
    data_received..............: 40 MB  312 kB/s
    data_sent..................: 45 MB  348 kB/s
  ✗ failed requests............: 3      0.023446/s
    http_req_blocked...........: avg=1.85ms   min=2µs      med=5µs     max=373.57ms p(90)=12.69µs p(95)=23.84µs
    http_req_connecting........: avg=1.11ms   min=0s       med=0s      max=47.62ms  p(90)=0s      p(95)=0s     
    http_req_duration..........: avg=2.65s    min=289.89ms med=2.48s   max=10.08s   p(90)=4.51s   p(95)=5.43s  
    http_req_receiving.........: avg=57.15ms  min=79µs     med=51.53ms max=200.89ms p(90)=81.12ms p(95)=91.61ms
    http_req_sending...........: avg=387.86µs min=159µs    med=332µs   max=3.87ms   p(90)=513.1µs p(95)=695µs  
    http_req_tls_handshaking...: avg=0s       min=0s       med=0s      max=0s       p(90)=0s      p(95)=0s     
    http_req_waiting...........: avg=2.59s    min=264.48ms med=2.4s    max=10.08s   p(90)=4.45s   p(95)=5.37s  
    http_reqs..................: 484    3.782635/s
    iteration_duration.........: avg=2.65s    min=290.24ms med=2.48s   max=10.08s   p(90)=4.51s   p(95)=5.43s  
    iterations.................: 484    3.782635/s
    vus........................: 22     min=1   max=22 
    vus_max....................: 100    min=100 max=100
```

In our use case, when reaching 22 virtual users (~3,7 requests per second), some incoming requests cannot be fulfilled before 10 seconds (`DEFAULT_WAIT_TIMEOUT` value).
During this test, CPU usage went from 0 to 100% and memory usage stayed low.

## Merge