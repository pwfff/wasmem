Compiled for Cloudflare Workers with https://github.com/golang/go/compare/master...twifkak:small

To deploy as a worker, first compile the WASM (using a go binary compiled from the above branch):
`GOOS=js GOARCH=wasm ~/src/go-small/bin/go build -a -o main.wasm src/wasmem/main.go`

Then generate the javascript:
`npm run build`

Then copy the dist/bundle.js to the workers console and add the WASM as a binding.
