# MultiVAC Main

## Installation

- Install Go. http://golang.org/doc/install

- Set `GOBIN` and `GOPATH` properly. Here is an example.

```bash
$ export GOPATH=$HOME/go
$ export GOBIN=$GOPATH/bin
$ export PATH=$PATH:$GOBIN
```

- Install glide (package management). https://github.com/Masterminds/glide (instructions copied below)

On Mac OS X you can install the latest release via [Homebrew](https://github.com/Homebrew/homebrew):

```
$ brew install glide
```

On Ubuntu install the package `golang-glide`:

```
$ sudo apt install golang-glide
```

- Install lint libs

```base
$ brew install golangci/tap/golangci-lint
$ go get -u golang.org/x/lint/golint
```

- Run the following commands to obtain MultiVAC and install it:

```bash
$ git clone https://github.com/multivactech/MultiVAC $GOPATH/src/github.com/multivactech/MultiVAC
$ cd $GOPATH/src/github.com/multivactech/MultiVAC
$ glide update && glide install
$ ./install
```

- A binary `MultiVAC` should be generated in `GOBIN`. Run it.

```bash
$ cd $GOBIN
$ MultiVAC
```

## Development

### Resolving dependencies

We use glide for dependency management. For the instruction on how to
install glide and update dependencies with glide, please see https://glide.sh/.

### Format your code

Call hooks/format.py to format your go code on any path under MultiVAC
code base before committing changes:

```bash
$ $MultiVAC_PATH/hooks/format.py
```

### Dependency injection

Check https://github.com/google/wire for detailed instruction.

The TL;DR instructions are:
- Run `go get github.com/google/wire/cmd/wire` and ensuring that $GOPATH/bin is added to your $PATH.
- Whenever you **add** a new injector file (with wire.Build, for example, adding a file like `sync/injector.go`), go
  to that package in command line, and run `wire`. Check if a generated `wire_gen.go` is generated in that package. 
  But ignore this step if you did not add any injector file, normally you don't need this step.
- Use `./install` in root package to install MultiVAC instead of mere "go install".

### Pre-submit

Install git hooks, you can trigger the following script from any path:

```bash
$ python $MultiVAC_PATH/hooks/install-hooks.py
```

If for any reason you don't want to run test for commit (such as in
your local branch), you can run `git commit --no-verify`. But please
specify the reason for skipping tests in commit message.

You can also run the pre-submit locally with command

```bash
$ $MultiVAC_PATH/hooks/pre-commit.py
```

### Unit test

You might find yourself in the need to mock some interfaces. There are certain open source tools to help mock in
golang easier. Such as mockery, you could follow the instructions in https://github.com/vektra/mockery. And you
could refer to controller_test.go in processor/controller as an example.


## Local end-to-end test

Check https://github.com/multivactech/testing-tools

## Resources

* CI: https://travis-ci.com/multivactech/MultiVAC/builds

