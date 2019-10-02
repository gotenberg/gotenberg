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

    done [==========================================================] 1m17.9s / 10m0s

    ✗ is status 200
     ↳  99% — ✓ 201 / ✗ 2
    ✗ is not status 504
     ↳  99% — ✓ 201 / ✗ 2
    ✓ is not status 500

    checks.....................: 99.34% ✓ 605   ✗ 4    
    data_received..............: 46 MB  594 kB/s
    data_sent..................: 9.6 MB 123 kB/s
  ✗ failed requests............: 2      0.025646/s
    http_req_blocked...........: avg=1.63ms   min=2µs   med=4µs     max=74.77ms  p(90)=9.6µs    p(95)=19.66ms 
    http_req_connecting........: avg=1.61ms   min=0s    med=0s      max=74.59ms  p(90)=0s       p(95)=19.57ms 
    http_req_duration..........: avg=2.43s    min=1.31s med=1.67s   max=10.27s   p(90)=5.44s    p(95)=8.75s   
    http_req_receiving.........: avg=97.95ms  min=103µs med=93.74ms max=202.48ms p(90)=114.51ms p(95)=132.99ms
    http_req_sending...........: avg=231.53µs min=92µs  med=201µs   max=2.33ms   p(90)=310.4µs  p(95)=333.59µs
    http_req_tls_handshaking...: avg=0s       min=0s    med=0s      max=0s       p(90)=0s       p(95)=0s      
    http_req_waiting...........: avg=2.33s    min=1.22s med=1.58s   max=10.12s   p(90)=5.33s    p(95)=8.62s   
    http_reqs..................: 203    2.603019/s
    iteration_duration.........: avg=2.43s    min=1.31s med=1.68s   max=10.27s   p(90)=5.45s    p(95)=8.75s   
    iterations.................: 203    2.603019/s
    vus........................: 13     min=1   max=13 
    vus_max....................: 100    min=100 max=100
```

In our use case, when reaching 13 virtual users (~2,6 requests per second), some incoming requests cannot be fulfilled before 10 seconds (`DEFAULT_WAIT_TIMEOUT` value).
During this test, CPU usage was high and memory usage went from 339 MiB to a peak of 421 MiB before going back to 364 MiB.

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

    done [==========================================================] 47.9s / 10m0s

    ✓ is not status 500
    ✗ is status 200
     ↳  83% — ✓ 31 / ✗ 6
    ✗ is not status 504
     ↳  83% — ✓ 31 / ✗ 6

    checks.....................: 89.18% ✓ 99    ✗ 12   
    data_received..............: 2.6 MB 54 kB/s
    data_sent..................: 3.4 MB 71 kB/s
  ✗ failed requests............: 6      0.125047/s
    http_req_blocked...........: avg=4.39ms   min=3µs   med=4µs    max=24.52ms p(90)=23.27ms  p(95)=23.62ms 
    http_req_connecting........: avg=4.34ms   min=0s    med=0s     max=24.42ms p(90)=23.15ms  p(95)=23.52ms 
    http_req_duration..........: avg=5.34s    min=1.74s med=4.33s  max=10.83s  p(90)=10.24s   p(95)=10.25s  
    http_req_receiving.........: avg=42.49ms  min=67µs  med=49.1ms max=68.87ms p(90)=57.34ms  p(95)=62.06ms 
    http_req_sending...........: avg=341.91µs min=215µs med=341µs  max=724µs   p(90)=443.99µs p(95)=514.79µs
    http_req_tls_handshaking...: avg=0s       min=0s    med=0s     max=0s      p(90)=0s       p(95)=0s      
    http_req_waiting...........: avg=5.3s     min=1.7s  med=4.27s  max=10.83s  p(90)=10.24s   p(95)=10.25s  
    http_reqs..................: 37     0.771121/s
    iteration_duration.........: avg=5.34s    min=1.74s med=4.35s  max=10.86s  p(90)=10.24s   p(95)=10.25s  
    iterations.................: 37     0.771121/s
    vus........................: 8      min=1   max=8  
    vus_max....................: 100    min=100 max=100
```

In our use case, when reaching 8 virtual users (~0.8 requests per second), some incoming requests cannot be fulfilled before 10 seconds (`DEFAULT_WAIT_TIMEOUT` value).
During this test, CPU usage was high and memory usage went from 315 MiB to a peak of 788 MiB before going back to 307 MiB.

## Merge

The Merge scenario is the same as the previous scenarios, but with a [gotenberg.pdf](../test/testdata/pdf/gotenberg.pdf) and a [gotenberg_bis.pdf](../test/testdata/pdf/gotenberg_bis.pdf).

```bash
$ k6 run --env MAX_VUS=100 --env BASE_URL=http://ec2-foo.eu-west-1.compute.amazonaws.com merge.js

          /\      |‾‾|  /‾‾/  /‾/   
     /\  /  \     |  |_/  /  / /    
    /  \/    \    |      |  /  ‾‾\  
   /          \   |  |‾\  \ | (_) | 
  / __________ \  |__|  \__\ \___/ .io

  execution: local
     output: -
     script: merge.js

    duration: -,  iterations: -
         vus: 1, max: 100

    done [==========================================================] 1m45.9s / 10m0s

    ✗ is status 200
     ↳  98% — ✓ 165 / ✗ 3
    ✗ is not status 504
     ↳  98% — ✓ 165 / ✗ 3
    ✓ is not status 500

    checks.....................: 98.80% ✓ 498   ✗ 6    
    data_received..............: 69 MB  649 kB/s
    data_sent..................: 70 MB  661 kB/s
  ✗ failed requests............: 3      0.028302/s
    http_req_blocked...........: avg=2.28ms   min=2µs      med=5µs      max=33.24ms  p(90)=23.6µs   p(95)=23.38ms 
    http_req_connecting........: avg=2.24ms   min=0s       med=0s       max=33.1ms   p(90)=0s       p(95)=23.04ms 
    http_req_duration..........: avg=5.32s    min=632.86ms med=5.24s    max=10.17s   p(90)=9.42s    p(95)=9.91s   
    http_req_receiving.........: avg=122.07ms min=82µs     med=114.72ms max=230.35ms p(90)=162.27ms p(95)=188.95ms
    http_req_sending...........: avg=16.56ms  min=349µs    med=1.37ms   max=223.25ms p(90)=12ms     p(95)=149.76ms
    http_req_tls_handshaking...: avg=0s       min=0s       med=0s       max=0s       p(90)=0s       p(95)=0s      
    http_req_waiting...........: avg=5.18s    min=573.03ms med=5.09s    max=10.13s   p(90)=9.29s    p(95)=9.78s   
    http_reqs..................: 168    1.58493/s
    iteration_duration.........: avg=5.33s    min=633.87ms med=5.24s    max=10.17s   p(90)=9.43s    p(95)=9.91s   
    iterations.................: 168    1.58493/s
    vus........................: 18     min=1   max=18 
    vus_max....................: 100    min=100 max=100
```

In our use case, when reaching 18 virtual users (~1.6 requests per second), some incoming requests cannot be fulfilled before 10 seconds (`DEFAULT_WAIT_TIMEOUT` value).
During this test, CPU usage was high and memory usage went from 310 MiB to a peak of 604 MiB before going back to 331 MiB.
