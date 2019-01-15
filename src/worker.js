import './wasm_exec'

let mod, bytes, inst;

addEventListener("fetch", function(event) {
    event.respondWith(handle(event.request));
}.bind(this));

function instantiate() {
    const go = new Go();

    inst = new WebAssembly.Instance(CYBERPUNK_WASM, go.importObject);
    go.run(inst);
}

async function handle(request) {
    try {
        if (typeof inst == 'undefined') {
            instantiate();
        }
        // Forward the request to our origin.
        let response = await fetch(request);

        // Check if the response is an image. If not, we'll just return it.
        // let type = response.headers.get("Content-Type") || "";
        // if (!type.startsWith("image/")) return response;

        // OK, we're going to resize. First, read the image data into memory.
        bytes = new Uint8Array(await response.arrayBuffer());

        // Call our WebAssembly module's init() function to allocate space for
        // the image.
        let pointer = initMem(bytes.length);

        let memoryBytes = new Uint8Array(inst.exports.mem.buffer);
        memoryBytes.set(bytes, pointer);

        let result = processImage();

        // arena could have moved
        memoryBytes = new Uint8Array(inst.exports.mem.buffer);
        // Extract the result bytes from WebAssembly memory.
        let resultBytes = memoryBytes.slice(result[0], result[0] + result[1]);

        // Create a new response with the image bytes. Our resizer module always
        // outputs JPEG regardless of input type, so change the header.
        let newResponse = new Response(resultBytes, response);
        newResponse.headers.set("Content-Type", "image/jpeg");

        // Return the response.
        return newResponse;
    } catch(err) {
        return new Response(err.stack)
    }
}
