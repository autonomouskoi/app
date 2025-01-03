# AutonomousKoi

This repo is for building the [AutonomousKoi](https://autonomouskoi.org) bot from source. It's
also a decent starting point for developking AK, but it's a little rough in that regard.

## Prerequisites

 - [Go](https://go.dev/) - At least 1.23
 - [Mage](https://magefile.org/) - Primary build tool
 - [NPM](https://www.npmjs.com/) - For building web content
 - [protoc](https://github.com/protocolbuffers/protobuf/releases) - AK uses protobuf generated code in Go and TypeScript

 ## Building for Mac and Linux

 This assumes you're building on Mac for Mac and on Linux for Linux.

 ```sh
 $ mage [releasemac|releaselinux]
 ```

 Mage will invoke the default target in `magefile.go`. This creates a directory called `work`.
 In `work`, some AutonomousKoi repos are cloned, `npm install` is run to pull in JS dependencies,
 and other `mage` build targets are invoked.

 If everything goes correctly, there will be a `work/dist` directory with either a `.dmg` or `.zip`
 file, for Mac or Linux respectively. Running the same `mage` command will skip many completed steps
 but will always recompile the executable and rebuild the package. `mage clean` will remove
 previously built artifacts.

 ## Building for Windows

 The prequisites and process are the same. However, I do this under [MSYS2](https://www.msys2.org/)
 to support binding with `libcrypto-3-x64.dll` which is required for `trackstarrekordbox`. I don't
 remember the steps to get my MSYS2 environment so if anyone wants to figure out the steps, I'd
 happily enshrine them here.

 ## Developing

 If you want to develop AK, you can `mage build` to do all the setup in `work`. If the repo you
 want to work on isn't in `work`, you can `git clone <repo>` and `go work use <dir>`.
