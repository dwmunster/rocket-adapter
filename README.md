<h1 align="center">Joe Bot - Rocket.Chat Adapter</h1>
<p align="center">Connecting joe with the Rocket.Chat chat application. https://github.com/go-joe/joe</p>
<p align="center">
	<a href="https://github.com/dwmunster/rocket-adapter/releases"><img src="https://img.shields.io/github/tag/dwmunster/rocket-adapter.svg?label=version&color=brightgreen"></a>
	<a href="https://circleci.com/gh/dwmunster/rocket-adapter/tree/master"><img src="https://circleci.com/gh/dwmunster/rocket-adapter/tree/master.svg?style=shield"></a>
	<a href="https://goreportcard.com/report/github.com/dwmunster/rocket-adapter"><img src="https://goreportcard.com/badge/github.com/dwmunster/rocket-adapter"></a>
	<a href="https://codecov.io/gh/dwmunster/rocket-adapter"><img src="https://codecov.io/gh/dwmunster/rocket-adapter/branch/master/graph/badge.svg"/></a>
	<a href="https://godoc.org/github.com/dwmunster/rocket-adapter"><img src="https://img.shields.io/badge/godoc-reference-blue.svg?color=blue"></a>
	<a href="https://github.com/dwmunster/rocket-adapter/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue.svg"></a>
</p>

---

This repository contains a module for the [Joe Bot library][joe].

**THIS SOFTWARE IS STILL IN ALPHA AND THERE ARE NO GUARANTEES REGARDING API STABILITY YET.**

## Getting Started

This library is packaged using the new [Go modules][go-modules]. You can get it via:

```
go get github.com/dwmunster/rocket-adapter
```

### Example usage

In order to connect your bot to rocket.chat you can simply pass it as module when
creating a new bot:

```go
package main

import (
	"os"

	"github.com/dwmunster/rocket-adapter"
	"github.com/go-joe/joe"
)

func main() {
	b := joe.New("example-bot",
		rocket.Adapter(
			os.Getenv("ROCKET_EMAIL"),
			os.Getenv("ROCKET_PASSWORD"),
			os.Getenv("ROCKET_URL"),
			os.Getenv("ROCKET_USER"),
		),
	)
	b.Respond("ping", Pong)

	err := b.Run()
	if err != nil {
		b.Logger.Fatal(err.Error())
	}
}

func Pong(msg joe.Message) error {
	return msg.RespondE("PONG")
}
```

So far the adapter will emit the following events to the robot brain:

- `joe.ReceiveMessageEvent`

## Built With

* [RocketChat/Rocket.Chat.Go.SDK](https://github.com/RocketChat/Rocket.Chat.Go.SDK) - Rocket.Chat API in Go
* [zap](https://github.com/uber-go/zap) - Blazing fast, structured, leveled logging in Go
* [pkg/errors](https://github.com/pkg/errors) - Simple error handling primitives

## Contributing

The current implementation is rather minimal and there are many more features
that could be implemented on the rocket.chat adapter so you are highly encouraged to
contribute. If you want to hack on this repository, please read the short
[CONTRIBUTING.md](CONTRIBUTING.md) guide first.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available,
see the [tags on this repository][tags. 

## Authors

- **Friedrich Große** - *Initial work on Slack adapter* - [fgrosse](https://github.com/fgrosse)
- **Drayton Munster** - *Conversion to Rocket.Chat API* - [dwmunster](https://github.com/dwmunster)

See also the list of [contributors][contributors] who participated in this project.

## License

This project is licensed under the BSD-3-Clause License - see the [LICENSE](LICENSE) file for details.

[joe]: https://github.com/go-joe/joe
[go-modules]: https://github.com/golang/go/wiki/Modules
[tags]: https://github.com/dwmunster/rocket-adapter/tags
[contributors]: https://github.com/github.com/dwmunster/rocket-adapter/contributors
