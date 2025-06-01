# traefik-http-cache-plugin
HTTP cache service developed as a Traefik plugin. 

### Parallel benchmarks
```
goos: darwin
goarch: arm64
pkg: github.com/Borislavv/traefik-http-cache-plugin/pkg/storage
cpu: Apple M2 Max
BenchmarkReadFromStorage

{"level":"info","time":1748771444,"message":"[0] Started BenchmarkReadFromStorage benchmark with 1 iterations."}
{"level":"info","time":1748771444,"message":"[0] TOTAL --->>> BenchmarkReadFromStorage total b.N: 1, total avg duration: 2025 ns/op"}

{"level":"info","time":1748771444,"message":"[1] Started BenchmarkReadFromStorage benchmark with 100 iterations."}
{"level":"info","time":1748771444,"message":"[1] TOTAL --->>> BenchmarkReadFromStorage total b.N: 100, total avg duration: 86 ns/op"}

{"level":"info","time":1748771444,"message":"[2] Started BenchmarkReadFromStorage benchmark with 10000 iterations."}
{"level":"info","time":1748771444,"message":"[2] TOTAL --->>> BenchmarkReadFromStorage total b.N: 10000, total avg duration: 92 ns/op"}

{"level":"info","time":1748771444,"message":"[3] Started BenchmarkReadFromStorage benchmark with 1000000 iterations."}
{"level":"info","time":1748771445,"message":"[3] TOTAL --->>> BenchmarkReadFromStorage total b.N: 1000000, total avg duration: 71 ns/op"}

{"level":"info","time":1748771445,"message":"[4] Started BenchmarkReadFromStorage benchmark with 18059074 iterations."}
{"level":"info","time":1748771470,"message":"[4] TOTAL --->>> BenchmarkReadFromStorage total b.N: 18059074, total avg duration: 30 ns/op"}

{"level":"info","time":1748771471,"message":"[5] Started BenchmarkReadFromStorage benchmark with 38933665 iterations."}
{"level":"info","time":1748771528,"message":"[5] TOTAL --->>> BenchmarkReadFromStorage total b.N: 38933665, total avg duration: 30 ns/op"}

BenchmarkReadFromStorage-12     	38933665	        31.24 ns/op	       8 B/op	       1 allocs/op
```
```
BenchmarkWriteIntoStorage

{"level":"info","time":1748771528,"message":"[1] Started BenchmarkWriteIntoStorage benchmark with 1 iterations."}
{"level":"info","time":1748771528,"message":"[1] TOTAL --->>> BenchmarkWriteIntoStorage total b.N: 1, total avg duration: 1854 ns/op"}
{"level":"info","time":1748771528,"message":"[2] Started BenchmarkWriteIntoStorage benchmark with 100 iterations."}
{"level":"info","time":1748771528,"message":"[2] TOTAL --->>> BenchmarkWriteIntoStorage total b.N: 100, total avg duration: 69 ns/op"}
{"level":"info","time":1748771528,"message":"[3] Started BenchmarkWriteIntoStorage benchmark with 10000 iterations."}
{"level":"info","time":1748771528,"message":"[3] TOTAL --->>> BenchmarkWriteIntoStorage total b.N: 10000, total avg duration: 94 ns/op"}
{"level":"info","time":1748771528,"message":"[4] Started BenchmarkWriteIntoStorage benchmark with 1000000 iterations."}
{"level":"info","time":1748771529,"message":"[4] TOTAL --->>> BenchmarkWriteIntoStorage total b.N: 1000000, total avg duration: 83 ns/op"}
{"level":"info","time":1748771529,"message":"[5] Started BenchmarkWriteIntoStorage benchmark with 15889005 iterations."}
{"level":"info","time":1748771538,"message":"[5] TOTAL --->>> BenchmarkWriteIntoStorage total b.N: 15889005, total avg duration: 110 ns/op"}

BenchmarkWriteIntoStorage-12    	15889005	        96.93 ns/op	      23 B/op	       1 allocs/op
```