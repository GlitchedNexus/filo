# Insights
This file contains all the things I've learnt over the project and I've add thse below for
future references and for those visiting to explore if it interests you. These are meant to
aid **MY** learning and may not be the comprehensive guide that you can refer to.

Topics that are included:
1. Interface syntax in go and implementing Interfaces
2. Struct syntax in Go
3. Documentation Practices
4. Best practices for developing maintainable and clean Go code.
5. Go Routines
6. Formatting strings in go
7. Anonymous function syntax
8. The GOB package
9. Defer
10. Select
11. Channels
12. TCP Handshakes
13. Blocking Calls
14. Go Routines
15. Problems associated with using interfaces as return values.
16. Uses cases for value method receivers andpointer method receivers.

## Interfaces
An interface in go is very similar to one in Java. It defines the functions that must be implmented by all types that adhere to the interface.

In Go, interfaces determine subtyping through a mechanism called "implementing" rather than explicit subtyping declarations. A type is considered to satisfy an interface if it provides all the methods defined by that interface.

## Structs
A little different from in syntax from Rust but like Rust all fields are private by default unless specified otherwise.
```Go
// Also notice the no `:` and `,` when defining the types of the struct fields.
type TCPTransport struct {
	ListenAddress string
	Listener      net.Listener

	mu sync.RWMutex
	peers map[net.Addr]Peer
}
```
To make a struct field public, it must start with an uppercase letter, which exports the field and allows access from outside the package.

## Documentation Practices
1. If you are documenting a type interface you should always start with the type as the first word.

## Best Pracitices
1. When defining fields, we use groups and put our mutexes above the things we want to protect.
  ```Go
  type TCPTransport struct {
  	ListenAddress string
  	Listener      net.Listener
  
  	// We group the mutex with the peers cause we want to protect it.
  	mu sync.RWMutex
  	peers map[net.Addr]Peer
  }
  ```
2. Functions names are in Pascal case. Public functions start with an uppercase letter while private functions start with a lowercase character.
3. For good quality code always put the public functions on top and private functions at the bottom.
Organize your code how someone is liekly to read it. Also put important functions on the top anbd helpers at the bottom.
4. Always make your errors lower caps.
5. When writing go libraries you should make an effort to provide NOP types that do nothing to allow easier testing.

## Go Routines


## Formatted strings
1. `%v` is used to print a value with its field names included when formatting structs
2. `%s` is placeholder for string values within a format string

## Anonymous functions
We can use the `func` to define anonymous functions like shown below.
```Go
	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("%+v\n", msg)
		}
	}()
```
Here the trailing `()` tells our code to “call it immediately”. So when we add the the trailing `()` it converts
the function value to function call without which we would get a compile time error.
```Go
go <function-call>

// Valid
go f()
go func() {}()

// Invalid
go f
go func() {}
```
In our invalid version the compiler sees: `go (a function value)`.

Go does not allow us to start a goroutine with a function value without calling it.

## The GOB Package
Helps to encode and decode binary values exhanged between the transmitter (Encoder) and Receiver (Decoder).

## Defer
The `defer` keyword schedules a function call to run when the surrounding function returns.
```Go
func example() {
    defer fmt.Println("world")
    fmt.Println("hello")
}
```
This code will give the output
```Text
hello
world
```
This is because when our function runs `fmt.Println("world")` is registered and it runs after example() finishes.

When Go sees:

```Go
defer f(x, y)
```

It does two things immediately:
1. Evaluates arguments (x, y)
2. Pushes the call onto a defer stack

Then later when the function returns (normally or via panic) all the deferred calls run in LIFO order (stack behavior).

## Select
The `select` keyword let's us wait on multiple channel operations and as often used in conjunction with go routines and channels.

## Channels
Channels are pipes that eneable communication between go routines. You can send values into channels from one goroutine and receive those values into another goroutine. To create a new channel we use
```Go
channel := make(chan <value-type>)
```

You can send using the `channel <- message` syntax. An example is included below
```Go
messages := make(chan string)
go func() {
	messages <- "ping"
}()
```

To read a message from the chanel we can do
```Go
messages := make(chan string)
go func() {
	messages <- "ping"
}()

// Read exactly one value from the channel into msg
msg := <- messages
fmt.Println(msg)
```

Having multiple goroutines read from a channel is a standard pattern and is safe. Each sent value goes to exactly one receiver (not all). All receivers “compete” for work and the Go runtime hands each message to one goroutine. An example of 
```Go
jobs := make(chan int)

for w := 0; w < 4; w++ {
    go func(id int) {
        for job := range jobs {
            fmt.Println("worker", id, "got job", job)
        }
    }(w)
}

for j := 0; j < 10; j++ {
    jobs <- j
}
close(jobs)
```


Many producers can send to one channel; the channel serializes the sends. An example of such a case is:
```Go
out := make(chan string)

go func() { out <- "a" }()
go func() { out <- "b" }()
fmt.Println(<-out)
fmt.Println(<-out)
```

## TCP Handshakes
When to TCP peers connect they send a handshake to acknowledege the connection, if the handshake is invalid or is not returned the connection is dropped.

## Blocking Calls
A blocking call is any operation where the current goroutine cannot continue executing until some condition is met and so it stops executing and waits.

When a goroutine hits a blocking call:
1. It stops execution
2. it uses no CPU
3. The Go scheduler runs other goroutines

These features make blocking cheap and are the reason why Go encourages
1. blocking channel reads
2. blocking mutex locks
3. simple synchronous-looking code

**NOTE: This is very different from busy-waiting.**

Blocking is also different from deadlocks as blocking is intentional and temporary whereas deadlocks are permanent blocking with no possible progress.

## Go Routines
Lightwieght concurrent execution units managed by the go runtime rather than the operating system and unlike
trational thread are cheap to create and efficient to schedule. They have a finite memory cost. 

The go runtime scheduler is repsonsible for multiplexing many go routines onto a small number of operating system threads.

When you launch go routines, you must consider their lifetimes and exit conditions. Without this you can get memory leaks, execessive memory consumption, deadlocks, etc.

Stages in a go routine lifetime:
1. We **spawn** a go routine by using the `go` keyword with a function call which starts execution asynchronously and may run concurrently with other go routines. 
2. The next stage is **execution** where it can be in a running state or blocked state. The go routine executes noramlly but may pause if it is waiting for a channel operation, is sleeping or waiting on a lock.
3. The last stage is **termination**. This can happen because a function it executes returns, the function encountered a fatal error or panic, the main function exits killing all remaining go routines or is explicitly cancelled using `context.Context`.

### Managing go routine lifetimes
1. Using `sync.WaitGroup` to wait for go routines to finish. The wait group helps to synchronize multiple go routines by keeping track of how many are still running 
  ```Go
  func worker(id int, wg *sync.WaitGroup) {
  	defer wg.Done() // Mark as done when finished.
  	fmt.Printf("Worker %d started \n", id)
  
  	time.Sleep(time.Second) // Simulate work
  	
  	fmt.Printf("Worker %d finished \n", id)
  }
  
  func main() {
  	var wg sync.WaitGroup
   
   	for i := 1; i <= 3; i++ {
    	wg.Add(1) // Increment counter
     	go worker(i, &wg)
    }
    
    wg.Wait() // Block until all workers are complete
    fmt.Println("All workers dones")
  }
  ```
2. Since go routines run independently and don't automatically stop when the main function exits. We can use `context.WithCancel` to allow controlled shutdown and tell go routines to exit.
  ```Go
  func worker(ctx context.Context) {
  	for {
  		select {
  			case <- ctx.Done():
  				fmt.Println("worker stopped")
  				return
  			default:
  				fmt.Println("workeing ...")
  				time.Sleep(500 * time.Millisecond)
  		}
  	}
  }
  
  func main() {
  	ctx, cancel := context.WithCancel(context.Background())
  
  	go worker(ctx) // Start worker go routine
  
  	time.Sleep(2 * time.Second)
  
  	cancel() // Stop worker go routine
  
  	time.Sleep(time.Second) // Give time for cleanup
  }
  ```
3. It is common practice to have go routines send data into channels that must have a corresponding receiver otherwise they will block indefinitely. We always to ensure that data is received and that channels are drained when you're done to avaoid go routine leaks.
  ```Go
  func worker(ch chan int) {
  	defer close(ch) // Close channel to prevent leaks
  
  	for i := 0; i< 5; i++ {
  		ch <- i // Send data
  	}
  }
  
  func main() {
  	ch := make(chan int)
  	go worker(ch)
  
  	for val := range ch {
  		fmt.Println(val) // Ensure data is received.
  	}
  }
  ```
4. We can also limit the number of go routines using worker pools since go routines do come with a memory cost.
  ```Go
  func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
  	defer wg.Done()
  	for job := range jobs {
  		fmt.Printf("Worker %d processing job %d\n", id, job)
  		time.Sleep(time.Second) // Simulate work
  		results <- job * 2
  	}
  }
  
  func main() {
  	const numWorkers = 3
  	jobs := make(chan int, 5)
  	results := make(chan int, 5)
  	var wg sync.WaitGroup
  
  	// Start worker goroutines
  	for i := 1; i <= numWorkers; i++ {
  		wg.Add(1)
  		go worker(i, jobs, results, &wg)
  	}
  
  	// Send jobs
  	for j := 1; j <= 5; j++ {
  		jobs <- j
  	}
  	close(jobs) // Close to signal no more jobs
  
  	wg.Wait()      // Wait for all workers to complete
  	close(results) // Close results channel
  
  	// Collect results
  	for res := range results {
  		fmt.Println("Result:", res)
  	}
  }
  ```

### Causes of race conditions
Races happen when goroutines also access shared memory (maps, slices, struct fields) without coordination. An example is included below:
```Go
counts := map[string]int{}
ch := make(chan string)

go func() {
    for s := range ch {
        counts[s]++ // ❌ map write from goroutine
    }
}()

go func() { ch <- "x" }()
go func() { ch <- "x" }()
```

### Avoiding race contions
The most common way to avoid such scenarios is to own the state in one goroutine. This means that we have one goroutine be the only one that mutates shared state. Everyone else communicates via channels. An example is given below:
```Go
type Event struct{ Key string }

events := make(chan Event)
done := make(chan struct{})

go func() {
    counts := map[string]int{}
    for e := range events {
        counts[e.Key]++
    }
    close(done)
}()

// many goroutines can send safely
go func() { events <- Event{"x"} }()
go func() { events <- Event{"x"} }()

close(events)
<-done
```
In this example, since only the “owner goroutine” touches counts → no locks needed.

Alternatively, we can use locks for shared memory if many goroutines must read / write the same structure.
An example is give below:
```Go
var mu sync.Mutex
counts := map[string]int{}

go func() {
    mu.Lock()
    counts["x"]++
    mu.Unlock()
}()
```

## Using Interfaces as return values
If we used the interface type (Transport) as a return value
we will encounter the problem of having to type cast
return value to a TCPTransport to get access to the
internal fields, i.e.,
```Go
func Test() {
  tcp := NewTCPTransport().(*TCPTransport)

  // We cannot call this without the type cast.
	tcp.listener.Accept()
}
```

## Defining struct methods

> **Use a pointer receiver (`*T`) whenever the method needs identity, mutation, or to avoid copying.
> Use a value receiver (`T`) only when the value is small and immutable.**

### Method receiver forms
Method receiver defining extra behaviours for the struct and can only be defined types that we own (defined in your package).
```go
func (m Model)  Foo() {}   // value receiver
func (m *Model) Bar() {}   // pointer receiver
```

### When to use a pointer receiver (`*Model`)

We use `*Model` if **any** of these are true

1. The method **mutates** the receiver
    ```go
    func (m *Model) Train() {
        m.weights = update(m.weights)
    }
    ```
    - If you used `(m Model)`, you’d mutate a **copy**.
2. The struct is **large or expensive to copy**
    ```go
    type Model struct {
        Weights []float64
        Cache   map[string][]float64
        Params  [4096]float64
    }
    ```
    - Value receiver would copy:
      * slice headers
      * maps
      * arrays
      * metadata
    - This is wasteful and it would be much cheaper to copy a pointer which is 8 bytes

3. The type has **internal identity / lifecycle**
    - Examples:
      * models
      * caches
      * connections
      * mutexes
      * file handles
    ```Go
    func (m *Model) Reset()
    func (m *Model) Close()
    ```
    - These are *stateful operations*.

4. Consistency: **if any method uses `*T`, all should**
    - Go style rule:
        > Don’t mix pointer and value receivers unless you have a very good reason.
    - This avoids:
        * confusing APIs
        * unexpected copying
        * interface mismatch

---

### When to use a value receiver (`Model`)

We use value receivers when **all** of the following are true
1. Method does **not mutate**
2. Struct is **small**
3. Struct is **logically immutable**

#### Examples of structs with value receiver methods

```go
type Point struct {
    X, Y float64
}

func (p Point) Norm() float64 {
    return math.Sqrt(p.X*p.X + p.Y*p.Y)
}
```

```go
type TimeRange struct {
    Start, End time.Time
}

func (t TimeRange) Duration() time.Duration {
    return t.End.Sub(t.Start)
}
```

### Go’s automatic dereferencing 

If you define:
```go
func (m *Model) Train() {}
```
You can call it on **either**:
```go
m := Model{}
m.Train()     // compiler takes &m automatically

mp := &Model{}
mp.Train()    // normal
```
But **we can NOT the other way around**, i.e., if you define:
```go
func (m Model) Train() {}
```
You **cannot** call it on a pointer method set in interfaces (see below).

### Method sets & interfaces
We know the different receiver types behave deifferently, we have summarized this below:
| Receiver | Method set contains                       |
| -------- | ----------------------------------------- |
| `Model`  | methods with `(Model)`                    |
| `*Model` | methods with `(Model)` **and** `(*Model)` |

This also means that for the interface
```go
type Trainer interface {
    Train()
}
```
It is totally valid for us to say
```go
func (m *Model) Train() {}

var t Trainer = &Model{} // ✅
```

But if Train uses *Model, we cannot say that
```go
var t Trainer = Model{}
```
This is because **Pointer receivers restrict interface satisfaction** This is intentional and good as it prevents accidental copying.

### Mutex rule
If your struct contains **any** of these:
1. `sync.Mutex`
2. `sync.RWMutex`
3. `sync.Once`
4. channels
5. maps that represent ownership

**We Always use pointer receivers** this is because copying these breaks correctness.

### Decision table for what receiver type to use.

| Question                          | Yes → | No →     |
| --------------------------------- | ----- | -------- |
| Does method mutate state?         | `*T`  | continue |
| Is struct large / owns data?      | `*T`  | continue |
| Constructor returns `*T`?         | `*T`  | continue |
| Has mutex / lifecycle / identity? | `*T`  | continue |
| Small, immutable, math-like?      |       | `T`      |


### Final TL;DR
> **Default to pointer receivers.
> Use value receivers only when you are very sure.**

1. Most real-world Go types use `*T`.
2. `*Model` → stateful, mutable, identity-based, non-trivial
3. `Model` → tiny, immutable, value-semantic
4. Constructor returns `&Model{}` → **methods should use `*Model`**
5. Go auto-derefs for you — ergonomics stay clean


## Why use Reader & Writers Instead of other things?
While using byte buffers is a valid option to send data between peers, readers and writers are much more general and are compatible with every data type.

# AES Encryption
