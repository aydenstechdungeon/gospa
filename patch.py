import re
import os

with open("gospa.go", "r") as f:
    gospa_content = f.read()

# 1. bytes.Buffer pool
gospa_content = gospa_content.replace(
    "var isrSemaphore = make(chan struct{}, 10)",
    "var isrSemaphore = make(chan struct{}, 10)\n\nvar bufferPool = sync.Pool{\n\tNew: func() interface{} {\n\t\treturn new(bytes.Buffer)\n\t},\n}"
)

# Replace buffer allocations
gospa_content = re.sub(
    r"var buf bytes\.Buffer\n\t+if rerr := ([^.]+)\.Render\([^,]+, &buf\)",
    r"buf := bufferPool.Get().(*bytes.Buffer)\n\tbuf.Reset()\n\tdefer bufferPool.Put(buf)\n\n\tif rerr := \1.Render(c.Context(), buf)",
    gospa_content
)
# handle buf.Bytes() -> buf.Bytes() (since it's a pointer now)
# Oh wait buf is a pointer now, so buf.Bytes() is still valid.

# 2. replace encoding/json with github.com/goccy/go-json
gospa_content = gospa_content.replace('"encoding/json"', '"github.com/goccy/go-json"')

# 3. Cache via config storage
# Just leaving it for a second. We can use a.Config.Storage
# Wait, let's look at website/main.go for client and server opts

with open("website/main.go", "r") as f:
    main_content = f.read()

main_content = main_content.replace('"encoding/json"', '"github.com/goccy/go-json"')

# Update Navigation and Compression options in main.go
# 4. WebSocket Binary Serialization -> MsgPack? Wait, we can use github.com/vmihailenco/msgpack/v5
# we can set Serializer and Deserializer on gospa.Config

cfg_repl = """  true,                // Compress WebSocket messages
g:          true,                // Only send state diffs
ableWebSocket:       true,
c(v interface{}) ([]byte, error) {
 msgpack.Marshal(v)
c(b []byte, v interface{}) error {
 msgpack.Unmarshal(b, v)
eed to ensure msgpack is imported
main_content = main_content.replace('"github.com/aydenstechdungeon/gospa"\n\t"github.com/aydenstechdungeon/gospa/routing"\n\t"github.com/gofiber/fiber/v3"\n)', '"github.com/aydenstechdungeon/gospa"\n\t"github.com/aydenstechdungeon/gospa/routing"\n\t"github.com/gofiber/fiber/v3"\n\n\t"github.com/vmihailenco/msgpack/v5"\n)')
main_content = main_content.replace("CompressState:         true,                // Compress WebSocket messages\n\t\tStateDiffing:          true,                // Only send state diffs\n\t\tEnableWebSocket:       true,", cfg_repl)

main_content = main_content.replace('HydrationMode:         "lazy",', 'HydrationMode:         "idle",')
main_content = main_content.replace('Enabled:        boolPtr(true),', 'Enabled:        boolPtr(true),\n\t\t\t\tIdleCallbackBatchUpdates: &gospa.NavigationIdleCallbackBatchUpdatesConfig{\n\t\t\t\t\tEnabled: boolPtr(true),\n\t\t\t\t},\n\t\t\t\tURLParsingCache: &gospa.NavigationURLParsingCacheConfig{\n\t\t\t\t\tEnabled: boolPtr(true),\n\t\t\t\t},')

with open("website/main.go", "w") as f:
    f.write(main_content)

with open("gospa.go", "w") as f:
    f.write(gospa_content)

