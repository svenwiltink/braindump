## Monkey patching net/http

### Why?
As a member and moderator of the Discord Gophers server we try to help
people with their questions about Go. Most of the time these questions
are fairly usual: How do goroutines work? Why does my application crash?
But over the last couple of months we have had new people come in and
ask if they can easily change the header order of net/http. The answer
has always been no, you would have to fork Go and change the net/http
library yourself. After a while the other Gophers in the server were
thinking of other possibilities. It has been on the back burner for a
while, until we read the excellent [blog post](https://bou.ke/blog/monkey-patching-in-go/) 
written by [Bouke](https://github.com/bouk) about monkey patching
patching in go.

### What is this monkey you speak of?
[Monkey patching](https://en.wikipedia.org/wiki/Monkey_patch) in short
is replacing existing code with something else while the application is 
running. In the ruby word, and in some other dynamic languages, this
is used to make testing easier in certain cases. It can also be used
to add new methods to existing classes. 

In Go this was thought to be impossible as it is a statically typed 
language. The language spec simply prevents us messing around with 
the code during runtime. Or so we thought! Bouke has found a way to do
this in Go by using a couple of clever tricks to change the function Go
executes when a function is called. The [blog post](https://bou.ke/blog/monkey-patching-in-go/) 
does a great job of explaining how it works, so I won't go into detail here.

### The hack part I
So Go can be monkey patched. Let's throw some patches at net/http and 
call it a day. There is no way to change the types of the header field.
It is defined as `type Header map[string][]string` so we'll have to
make that work. [Doad](https://github.com/zacharyburkett) came up with the idea
of using a single entry in the header map to store the other headers as
well. If the 'single' header had newlines in them they would be properly
sent as multiple headers. This normally won't work as newlines are not
allowed in the header value. net/http will return an error if
you try:

<script src="https://gist.github.com/svenwiltink/8e592735143e4d665790ce33a3250fc6.js"></script>


```text
Get "https://sven.wiltink.dev": net/http: invalid header field value "SomeValue\
nOtherHeader: OtherValue" for key SomeHeader
```

This pesky validation has to go away! It is performed in the [Transport layer](https://github.com/golang/go/blob/b1be1428dc7d988c2be9006b1cbdf3e513d299b6/src/net/http/transport.go#L514
) by calling httpguts.ValidHeaderFieldValue. The patch target has found!
Trying to patch this results in the following code:

<script src="https://gist.github.com/svenwiltink/8850a82a12460e3efb658b0def752bc1.js"></script>

The patch is in place and the code run, disappointingly yielding the same
error.

```text
Get "https://sven.wiltink.dev": net/http: invalid header field value "SomeValue\
nOtherHeader: OtherValue" for key SomeHeader
```

This doesn't add up, the function was patched! So why isn't the patched function called?
By adding this snippet of code in net/http and our main we can see the function pointers
are in fact different:
```go
fmt.Printf("pointer in net/http: %d\n", reflect.ValueOf(httpguts.ValidHeaderFieldValue).Pointer())
fmt.Printf("pointer in main: %d\n", reflect.ValueOf(httpguts.ValidHeaderFieldValue).Pointer())
```
```
pointer in main: 6554096
pointer in net/http: 6228976
```

### The detective work
Somehow the standard library calls a different 'instance' of the function that we are trying
to patch. Which one is it and why does it have a different address? Using `readelf` we can dump
the symbols in the binary. After converting the pointer to hex I found the following:
````go
readelf -a -W banaan | grep -i 5f0bf0     
  6610: 00000000005f0bf0    60 FUNC    GLOBAL DEFAULT    1 vendor/golang.org/x/net/http/httpguts.ValidHeaderFieldValue
````
net/http is calling a function that is prefixed with "vendor". It turns out Go vendors the 
/x/ packages it needs in the standard library. The function we are patching isn't the
same 'instance' of the function. The vendored version can be found [here](https://github.com/golang/go/tree/c5cf6624076a644906aa7ec5c91c4e01ccd375d3/src/vendor/golang.org/x/net/http/httpguts).
We don't normally have access to this function from within our application, but there is 
a hacky way.

### Linkname enters the chat
The go compiler toolchain has a special tool that enables us to define a function link it
to a different implementation. This is called `linkname` and is used by some parts of the standard
library to call functions without importing a package. An example of this is the sync package.
The sync package wants call the runtime function `nanotime` but because of import restrictions
it can't import the runtime package directly. Instead the sync package [defines a function stub](https://github.com/golang/go/blob/0a820007e70fdd038950f28254c6269cd9588c02/src/sync/runtime.go#L57)
and the runtime package uses `linkname` [to link the two functions together](https://github.com/golang/go/blob/0a820007e70fdd038950f28254c6269cd9588c02/src/runtime/sema.go#L614).

The behaviour of `linkname` and other `//go:` comments is explained in the compile command documentation: https://golang.org/cmd/compile/
```go
//go:linkname localname [importpath.name]
```
> This special directive does not apply to the Go code that follows it. Instead, the //go:linkname directive instructs the compiler to use “importpath.name” as the object file symbol name for the variable or function declared as “localname” in the source code. If the “importpath.name” argument is omitted, the directive uses the symbol's default object file symbol name and only has the effect of making the symbol accessible to other packages. Because this directive can subvert the type system and package modularity, it is only enabled in files that have imported "unsafe".

In the previous section the name of the vendored function function was discovered. `linkname` can
be used to 'magically' have access to it. Using the code below we can demonstrate the pointer of this
new function is the same as the one used by `net/http`:

<script src="https://gist.github.com/svenwiltink/f6deda1983c555a14032cf0f72e77501.js"></script>
```text
pointer in main: 6228976
pointer in net/http: 6228976
```

There is now a function called `linknameMagic` that shares the function pointer of the vendored
function! Because `linknameMagic` is no different from other functions from the perspective of
our code it can also be monkey patched:

<script src="https://gist.github.com/svenwiltink/423ca78638668eb46c7e97dfd64973f2.js"></script>

```text
pointer in main: 6228976
pointer in net/http: 6228976
the patched function was called!
the patched function was called!
the patched function was called!
the patched function was called!
the patched function was called!
<nil>
```

Note how performing the request also does not return an error anymore. The `net/http` library
has been fooled! When inspecting the http request the headers are however still on a single line 
and not newline separated as they should. This is because [the transport code of net/http](https://github.com/golang/go/blob/0a820007e70fdd038950f28254c6269cd9588c02/src/net/http/header.go#L186)
replaces all newlines with spaces. There is nothing that stops us from monkey patching that
as well, but I will be leaving that as an exercise for the reader.


Signing off,  
Sgt_Tailor, aka CursedLinkname