# Contributing

## Contributing code changes

1. fork the repo
2. use `make build` for a first build
3. start a branch and hack your changes
4. use `make test` to test your changes
5. send a pull request
6. glory, fame, money and glory (yes, twice)

## IDE settings

1. Make sure your IDE runs `gofmt -w -s` on file save.

2. Make sure your IDE adds a blank line at the end of the file.

### VSCode

1. To format on save, install the [Go extension for VSCode](https://code.visualstudio.com/docs/languages/go) and go to settings:
    1. in `Go: Format Flags` add `-w -s`
    1. in `Go: Format Tools` select `gofmt`

2. To insert a new line at the end of file on save, go to settings:
    1. in `Files: Insert Fina lNewline` mark the checkbox
