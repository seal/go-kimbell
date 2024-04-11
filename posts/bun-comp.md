
# Performance Benchmarking of Web Runtimes: Go, Rust, Bun, and Node.js

## Introduction

In the ever-evolving landscape of software-development, many get stuck in the loop of what language/framework to choose. I myself had this issue when I started coding, whether to start with C, C++, Java etc. Whilst I over-came this problem, the question of which is best (fastest, dur) never left my mind. Whilst this post may contribute to your inability to choose, it was fun to make .

Go, Rust, Bun, and Node.js. Our goal was to determine their capabilities in handling HTTP requests and JSON serialization. This test was inspired by the recent release of Bun v1.0, which showed promise as a resource-efficient runtime.

Please note I have taken heavy inspiration from this video https://www.youtube.com/watch?v=yPcWzSlsteA and this blog post https://www.priver.dev/blog/benchmark/go-vs-rust-vs-bun-vs-node-http-benchmark/ ( check them out)

## Testing Environment
Our benchmark tests were conducted locally and requests were made to localhost:3000


Local machine specs:
```
AMD Ryzen 7 5800X (16) @ 3.800GHz 
32GB ram @ 2800 Mhz
Some wonderful M.2 drive ( https://documents.westerndigital.com/content/dam/doc-library/en_us/assets/public/western-digital/product/internal-drives/pc-sn720-ssd/data-sheet-pc-sn720-compute.pdf)
```
The software versions are as follows:

```
- Rust: rustc 1.74.0-nightly (bdb0fa3ee 2023-09-19)
- Go: go version go1.21.1 linux/amd64
- Bun: 1.0.2
- Node.js: v20.6.1
```

## Testing Methodology
We conducted one type of test initially:

1. Take a simple request, with 4 query parameters
2. Add them to an object
3. "Marshal" / Serialise them 
4. Write it to file ( file name = uuid v4)
5. Read from said file
6. Return the file response 

## Running the tests



Whilst I could individually run said tests one by one, I wanted to record metrics for the specific time-frame that the test is running and record these into a graph 

As rust has piqued my interest I chose it as my language of choice, and ran the following program ( check it out I'm quite proud of it) https://github.com/seal/speed-test/blob/main/graphs/src/main.rs

The method was simple:
1. Start build steps if there are any 
2. Start the server
3. Start the wrk benchmarking command ( 300s, 12 threads, 400 conns)
WRK Command :
```rust
  TestConfig {
            name: "bun".to_string(),
            wrk_args: vec![
                "-t12",
                "-c400",
                "-d300s",
                "http://127.0.0.1:3000/?q1=1&q2=2&q3=3&q4=4",
            ]
            .into_iter()
            .map(|s| s.into())
            .collect(),
            script_dir: dir_path_to_string("bun"),
            script_args: vec!["index.ts"].into_iter().map(|s| s.into()).collect(),
            build_step_command: None,
            command: "bun".to_string(),
            build_step_args: None,
        },

```

4. Record CPU and RAM values and record into a graph to be later viewed 
5. Drop the server, wait approx 15 seconds for i/o to be stopped ( Go and Rust don't like to shut-down immediately, sending a pkill/kill command works, but we need to wait for i/o anyway)
6. Rinse and repeat 
## Local Test Results

The full wrk outputs are in the github repo, link here:
https://github.com/seal/speed-test/tree/main
But the main result we're looking at is reqs/s and total

Results: ( CPU = RED, RAM = BLUE)
```
Bun  - 1.97k/s - 7041644
```

![](https://github.com/seal/speed-test/blob/main/results/local/initial_bun_usage.png?raw=true)

```
Go   - 4.33k/s - 15510114
```

![](https://github.com/seal/speed-test/blob/main/results/local/go_usage.png?raw=true)
```
Rust - 5.20k/s - 18615033
```

![](https://github.com/seal/speed-test/blob/main/results/local/rust_usage.png?raw=true)
```
Node - 2.22k/s - 7954136
```

![](https://github.com/seal/speed-test/blob/main/results/local/node_usage.png?raw=true)

Yes, you read that right, node was *faster* than bun

Why ? 

I only had one possible answer, query parameters 
```javascript
const { searchParams } = new URL(req.url)
const id = uuidv4();
const responseJson = {
	q1: searchParams.get("q1"),
	q2: searchParams.get("q2"),
	q3: searchParams.get("q3"),
	q4: searchParams.get("q4"),
};
```

I decided to change this code to 
```javascript 
const parsedUrl = url.parse(req.url);
const queryParamsString = parsedUrl.query;
const queryParams = querystring.parse(queryParamsString);
const id = uuidv4();
const responseJson = {
	q1: queryParams.q1,
	q2: queryParams.q2,
	q3: queryParams.q3,
	q4: queryParams.q4,
};
```
and re-run the test...

The result? 
```
Running 5m test @ http://127.0.0.1:3000/?q1=1&q2=2&q3=3&q4=4
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    20.22ms    1.41ms  48.26ms   76.61%
    Req/Sec     1.64k    96.88     3.59k    84.79%
  5874953 requests in 5.00m, 857.23MB read
Requests/sec:  19579.00
Transfer/sec:      2.86MB
```

*It got worse*
Now here I'm a little confused, so I decided to re-run all of the tests again, thinking the 2% cpu usage from spotify might be affecting my results, or possibly node / rust / go was not shutting down properly, I also moved bun to *run first*

Bun:
```
Running 5m test @ http://127.0.0.1:3000/?q1=1&q2=2&q3=3&q4=4
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    20.17ms    1.36ms  46.92ms   77.03%
    Req/Sec     1.64k    93.89     2.00k    85.22%
  5890346 requests in 5.00m, 859.47MB read
Requests/sec:  19630.93
Transfer/sec:      2.86MB
```

Node:
```
Running 5m test @ http://127.0.0.1:3000/?q1=1&q2=2&q3=3&q4=4
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    14.92ms    3.53ms 344.78ms   99.72%
    Req/Sec     2.23k    79.48     3.03k    76.54%
  7996381 requests in 5.00m, 1.57GB read
Requests/sec:  26648.65
Transfer/sec:      5.36MB
```


Well, that's weird, being myself not a javascript programmer I initially put this down to bad code.
Along with this, bun was not deleting all the files? 
I was left with 100-200 files remaining after the program had completed it's run.

Now my only conclusion left was *bun was bad at query parameters regardless of library / method*

I removed the need for query parameters and just making the request to localhost:3000/ then re-ran my tests
Bun:
```
Running 5m test @ http://127.0.0.1:3000/
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    15.71ms    1.23ms  38.00ms   75.57%
    Req/Sec     2.11k   156.77     3.04k    67.74%
  7562611 requests in 5.00m, 1.08GB read
Requests/sec:  25204.39
Transfer/sec:      3.68MB
```

![](https://github.com/seal/speed-test/blob/main/results/local/bun_no_params_usage.png?raw=true)
Node:
```
Running 5m test @ http://127.0.0.1:3000/
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    15.24ms    3.32ms 336.85ms   99.81%
    Req/Sec     2.18k    73.56     3.18k    71.07%
  7826277 requests in 5.00m, 1.54GB read
Requests/sec:  26082.39
Transfer/sec:      5.25MB
```

![](https://github.com/seal/speed-test/blob/main/results/local/node_no_params_usage.png?raw=true)


Bun was slower, *again*

I then re-made the tests to do a 30 second test, as due to the ram usage I assumed this was a garbage collector issue 

The result ? 
Bun:
```
Running 30s test @ http://127.0.0.1:3000/
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    15.49ms    1.17ms  34.88ms   74.94%
    Req/Sec     2.14k   158.38     2.33k    41.53%
  767184 requests in 30.02s, 111.94MB read
Requests/sec:  25557.37
Transfer/sec:      3.73MB
```

![](https://github.com/seal/speed-test/blob/main/results/local/30s_bun_no_params.png?raw=true)
Node:
```
Running 30s test @ http://127.0.0.1:3000/
  12 threads and 400 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    16.56ms   10.98ms 350.33ms   99.46%
    Req/Sec     2.08k   136.75     3.22k    92.52%
  744045 requests in 30.01s, 149.72MB read
Requests/sec:  24793.63
Transfer/sec:      4.99MB
```

![](https://github.com/seal/speed-test/blob/main/results/local/30s_node_no_params.png?raw=true)


## Code 
All the code from this project can be found here
https://github.com/seal/speed-test
Please critique this code, as I believe my test results may be skewed due to incorrect code.
## Conclusion

Our benchmarking results reveal some interesting things, rust at #1, go close #2 
But interestingly enough bun does not perform as well as node in some situations ( most from my tests)

Whilst I chose a more extensive testing method than others, I believe there could be some issues with my bun code leading to these results.

All in all, rust will continue to be used for my personal projects, Go for clients and I'm going to continue to do this sort of testing on new "faster" frameworks that collect all of the hype.


## Feedback

I welcome feedback and discussion on these findings. You can reach out to me on Twitter at [https://twitter.com/bytebitter](https://twitter.com/bytebitter) for any questions or comments.

