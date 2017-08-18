# Profile Agent

## Intro

 Made to perform profiling on live enviroments of application and push details on promotheus.

 ```go run executable/main.go ``` to see the demo from ```profile-agent``` directory. and hit http://localhost:8081/measure-func url to hit memory-leak simulation for 2 minutes. You can see in console for the list of files, as well if you point prometheus server on http://localhost:8081/metrics, and query for 'profileagent_example_memory_histo_sample_bucket', you will find list of functions and amount of memory leak.
```
[
  924248byte
  0
  1503065445
  &{
    root
    2.401537e+06
    1
    []
    map[
      github.com/darshanman/profile-agent/examples.leakMemory.func1(/.../.../github.com/darshanman/profile-agent/examples/app.go:56):0xc4201623c0
    ]
    0xc420298740
  }
]

```


 ### Current:
 - working to identify memory leaks

 ### Future:
- Need to add for cpu profiling,
- code clean up
