# Go Logger

Logger is a powerful and customizable logging library for Go applications. It provides advanced logging features such as log rotation, configurable log levels, and multiple output formats. Logger is designed to be easy to integrate into your projects, offering a simple and consistent API for managing logs.

## Features

- Log rotation: Automatically rotate log files based on size or time
- Configurable log levels: Trace, Debug, Info, Warning, Error, Fatal, and Panic
- Multiple output formats: Console, JSON
- Easy integration with third-party log shippers and analyzers
- Highly customizable: modify log format, timestamp format, and more
- Efficient and lightweight

## Installation

To install Logger, use `go get`:

```bash
go get -u github.com/yourusername/logger
```


## Usage

Import the package in your Go project:

```go
import "github.com/imkiptoo/logger"
```

Create a new logger instance:

```go
log := logger.New(logger.Config{
    Level:       logger.InfoLevel,
    Output:      logger.ConsoleOutput,
    TimeFormat:  logger.DefaultTimeFormat,
    LogFormat:   logger.DefaultLogFormat,
    Rotate:      true,
    MaxSize:     10 * 1024 * 1024,
    MaxAge:      7,
    MaxBackups:  3,
})
```

Use the logger instance to log messages:
    
```go
log.Info("This is an info message")
log.Warn("This is a warning message")
log.Error("This is an error message")
```

## Configuration

You can customize the behavior of the logger by providing a `logger.Config` struct when creating a new logger instance. The following options are available:

- `Level`: Set the minimum log level (default: `logger.InfoLevel`)
- `Output`: Set the output format (default: `logger.ConsoleOutput`)
- `TimeFormat`: Set the format for timestamps (default: `logger.DefaultTimeFormat`)
- `LogFormat`: Set the format for log messages (default: `logger.DefaultLogFormat`)
- `Rotate`: Enable or disable log rotation (default: `true`)
- `MaxSize`: Set the maximum size for log files (default: 10 MB)
- `MaxAge`: Set the maximum age for log files in days (default: 7)
- `MaxBackups`: Set the maximum number of backup files to keep (default: 3)

## Contributing

We welcome contributions from the community! Please submit any bug reports, feature requests, or pull requests to the GitHub repository.

Before submitting a pull request, please make sure you have:

- Forked the repository
- Created a new branch for your changes
- Committed your changes with clear and descriptive commit messages
- Run tests to ensure your changes do not break existing functionality
- Updated documentation, if necessary

Once your pull request is submitted, a maintainer will review your changes and provide feedback. Please be patient, as this process may take some time.

## License

Logger is released under the MIT License. Below is the full text of the license:

MIT License

Copyright (c) 2023 Isaac Kiptoo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
