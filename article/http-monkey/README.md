## Monkey patching net/http

### Why?
As a member and moderator of the Discord Gophers server we try to help
people with their question about Go. Most of the time these questions
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
does a great job of explaining how it works so I won't go into detail here.

### The hack part I
So Go can be monkey patched. Let's throw some patches at net/http and 
call it a day. There is no way to change the types of the header field.
It is defined as `type Header map[string][]string` so we'll have to
make that work. <insert github username (doad)> came up with the idea
of using a single entry in the header map to store the other headers as
well. If the 'single' header had newlines in them they would be properly
sent as multiple headers. This normally won't work as newlines aren't
are not allowed in the header value. net/http will return an error if
you try:

```text
Get "https://sven.wiltink.dev": net/http: invalid header field value "SomeValue\
nOtherHeader: OtherValue" for key SomeHeader
```

This pesky validation has to go away! It is performed in the [Transport layer](https://github.com/golang/go/blob/b1be1428dc7d988c2be9006b1cbdf3e513d299b6/src/net/http/transport.go#L514
) by calling httpguts.ValidHeaderFieldValue. The patch target has found!
Trying to patch this results in the following code:

<link to file>

The patch is in place and the code run, disappointingly yielding the same
error.