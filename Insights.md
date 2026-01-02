# Interfaces
An interface in go is very similar to one in Java. It defines the functions that must be implmented by all types that adhere to the interface.

In Go, interfaces determine subtyping through a mechanism called "implementing" rather than explicit subtyping declarations. A type is considered to satisfy an interface if it provides all the methods defined by that interface.

# Structs
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

# Documentation Practices
1. If you are documenting a type interface you should always start woth the type as the first word.

# Best Pracitices
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

# Go Routines


# Formatted strings
1. `%v` is used to print a value with its field names included when formatting structs
2. `%s` is placeholder for string values within a format string

# Anonymous functions
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

# The GOB Package
Helps to encode and decode binary values exhanged between the transmitter (Encoder) and Receiver (Decoder).

# Defer
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

# Select

# Channels

# Handshakes
