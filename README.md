# bast

## A lightweight RESTful  for Golang


> Install

``` bash

 go get -u github.com/aixiaoxiang/bast

 ```

# Router doc(Request example)

> Router
 

### Get

``` golang
//Person struct
type Person struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

bast.Get("/xxx", func(ctx *bast.Context) {
    //verify imput parameter
    err := ctx.Verify("name=required|min:1", "age=required|min:1")
    if err != nil {
        ctx.Failed(err.Error())
        return
    }

    name := ctx.GetString("name")
    age := ctx.GetInt("age")

    //handling
    //...

    person := &Person{
        Name: name,
        Age:  Age,
    }
    //handling
    //...
    ctx.JSON(person)
}) 
```
 

### Post

``` golang 
//Person struct
type Person struct {
    Name string `json:"name" v:"required|min:1"`
    Age  int    `json:"age"  v:"min:1"`
}

bast.Post("/xxx", func(ctx *bast.Context) {
    person := &Person{}
    err := ctx.JSONObj(person) //or ctx.JSONObj(person,true) //version of verify imput parameter
    if err != nil {
        ctx.Failed("sorry! invalid parameter")
        return
    }
    person.Age += 2

    //handling
    //...

    ctx.JSON(person)
})
    
```

### Run 

``` golang

bast.Run(":9999")

```
  

# CommandLine

` Like nginx commandline `

### If Your program name is ``` Ai ```

#### -h | -help

``` bash

    ./Ai -h

```

#### -start   

` Run in background  `

``` bash

    ./Ai -start

```

#### -stop 

` stop program `

``` bash

    ./Ai -stop

```

#### -reload    

`graceful restart. stop and start`

``` bash

    ./Ai -reload

```

#### -conf 

` seting config files.(default is ./config.conf)`

``` bash

    ./Ai -conf=your path/config.conf 

```


#### -install 

`installed as service.(daemon) `


``` bash

    ./Ai -install

```


#### -uninstall 

`uninstall a service.(daemon) `


``` bash

    ./Ai -uninstall

```
 

#### -migration 
 
` migration or initial system(handle sql script ...) `

``` bash

    ./Ai -migration

```
 
### Such as

>` run program (run in background) `


``` bash  

    ./Ai -start -conf=./config.conf 

```


> ` deploy program (startup) `


``` bash  

    ./Ai -install

```

# config template

` support multiple instances` 
 

``` json
[
    {//a instance
        "key":"xxx-conf",
        "name":"xx",
        "addr":":9999",
        "fileDir":"./file/",//(default is ./file/)
        "debug":false,
        "baseUrl":"",
        "log":{
            "outPath":"./logs/logs.log", //(default is ./logs/logs.log)
            "level":"debug",
            "maxSize":10,
            "maxBackups":3,
            "maxAge":28,
            "debug":false,
            "logSelect":false,
        },
        "conf":{//user config(non bast framework)
            "key":"app",
            "name":"xxx",
            "dbTitle":"xxx",
            "dbName":"xxxx",
            "dbUser":"xxx",
            "dbPwd":"xxx",
            "dbServer":"localhost"
            //..more field..//
        }
    }
    //..more instances..//
]

```

# Distributed system unique ID    

> [snowflake-golang](https://github.com/bwmarrin/snowflake)  or [snowflake-twitter](https://github.com/twitter/snowflake)   
 

> use  

``` golang

  id := bast.ID()
  fmt.Printf("id=%d", id)

```

> benchmark test ``` go test  -bench=. -benchmem  ./ids```   
physics cpu ``` 4 ```

``` bash

    go test   -bench=. -benchmem  ./ids
    goos: darwin
    goarch: amd64 
    Benchmark_ID-4              20000000    72.1 ns/op       16 B/op     1 allocs/op
    Benchmark_Parallel_ID-4     10000000    150 ns/op        16 B/op     1 allocs/op
    PASS
    ok      github.com/aixiaoxiang/bast/ids 10.126s

```
