# Kubernetes Port Forwarding in Go — How It Actually Works

A practical guide to implementing programmatic Kubernetes port forwarding in Go
using `client-go`. Covers the protocol, the libraries, pod resolution strategies,
reconnection logic, and the gotchas you'll hit along the way.

## The Protocol: SPDY Under the Hood

`kubectl port-forward` isn't doing anything magical. Under the hood, Kubernetes
port forwarding uses SPDY (an HTTP/2 predecessor) to multiplex data streams over
a single connection to the API server. The flow is:

1. Your client sends a `POST` to the pod's `portforward` subresource:
   ```
   POST /api/v1/namespaces/{ns}/pods/{pod}/portforward
   ```
2. The API server upgrades the HTTP connection to SPDY.
3. Two SPDY streams are opened per forwarded port — one for data, one for errors.
4. The kubelet on the node proxies traffic between your SPDY streams and the
   target container port.

This is important to understand because it means: your traffic always routes
through the API server and kubelet. It is not a direct TCP connection to the pod.
Latency and throughput reflect that extra hop.

## Libraries You Need

All of these come from `k8s.io/client-go`:

| Package | Role |
|---|---|
| `k8s.io/client-go/tools/portforward` | The `PortForwarder` — manages the SPDY streams and local listener |
| `k8s.io/client-go/transport/spdy` | `RoundTripperFor()` creates the SPDY transport; `NewDialer()` creates the upgrade dialer |
| `k8s.io/client-go/rest` | The `*rest.Config` that carries cluster auth, TLS certs, etc. |

You'll also need `k8s.io/client-go/kubernetes` (or the dynamic client) to build
the portforward URL and to resolve pods.

## Minimal Working Example

```go
import (
    "fmt"
    "io"
    "net/http"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/portforward"
    "k8s.io/client-go/transport/spdy"
)

func forward(config *rest.Config, clientset *kubernetes.Clientset,
    namespace, podName string, localPort, remotePort int,
    stopCh, readyCh chan struct{},
) error {
    // 1. Build the URL to the pod's portforward subresource
    url := clientset.CoreV1().RESTClient().Post().
        Resource("pods").
        Namespace(namespace).
        Name(podName).
        SubResource("portforward").
        URL()

    // 2. Create the SPDY transport + dialer
    transport, upgrader, err := spdy.RoundTripperFor(config)
    if err != nil {
        return fmt.Errorf("creating SPDY transport: %w", err)
    }
    dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

    // 3. Create the port forwarder
    ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
    fw, err := portforward.New(dialer, ports, stopCh, readyCh, io.Discard, io.Discard)
    if err != nil {
        return fmt.Errorf("creating forwarder: %w", err)
    }

    // 4. Blocks until stopCh is closed or the tunnel drops
    return fw.ForwardPorts()
}
```

Key points:
- `stopCh` — close this channel to tear down the tunnel.
- `readyCh` — the forwarder closes this once the local listener is bound and
  ready to accept connections. Don't send traffic before this fires.
- The last two args to `portforward.New()` are `io.Writer` for stdout/stderr from
  the tunnel. In production, pipe these to your logger or discard them.

## Auto-Assigning a Local Port

Pass `0` as the local port and the OS picks a free one. After `readyCh` fires,
call `fw.GetPorts()` to find out what was assigned:

```go
go func() {
    <-readyCh
    ports, _ := fw.GetPorts()
    if len(ports) > 0 {
        fmt.Printf("listening on 127.0.0.1:%d\n", ports[0].Local)
    }
}()
```

**Gotcha:** `GetPorts()` returns empty until the tunnel is actually established.
Only call it after `readyCh` is closed.

## Bridging `context.Context` and `stopCh`

`client-go`'s portforwarder predates Go contexts — it uses raw channels.
You'll almost certainly want to bridge the two:

```go
stopCh := make(chan struct{})
go func() {
    <-ctx.Done()
    close(stopCh)
}()
```

Don't close `stopCh` from multiple goroutines — closing an already-closed channel
panics. The context bridge pattern above is safe because `ctx.Done()` only fires
once.

## Resolving What to Forward To

Users think in terms of Services and Deployments. Kubernetes port forwarding only
works against **pods**. You need a resolution layer.

### Direct Pod Target

Trivial — use the pod name as-is. But consider: when that pod dies, the forward
is dead with no way to recover. This is fine for debugging; it's bad for a
persistent tool.

### Selector-Based Resolution (Services, Deployments, StatefulSets)

The more robust approach:

1. Fetch the parent resource (Service, Deployment, etc.).
2. Extract its label selector:
   - **Services** use `spec.selector` (a flat `map[string]string`).
   - **Deployments/StatefulSets** use `spec.selector.matchLabels`.
3. List pods matching those labels.
4. Pick the best candidate.

**Pod ranking heuristic:** When multiple pods match, prefer:
1. Pods in `Running` phase over non-running.
2. Pods with the `Ready` condition over not-ready.
3. Newest creation timestamp as a tiebreaker (most likely to be up-to-date
   during a rolling update).

**Why not just pick a random running pod?** During rolling updates, you have both
old and new pods running simultaneously. Preferring the newest ready pod means
your forward follows the rollout naturally.

### Using the Dynamic Client

If you want to support arbitrary resource types without hardcoding each one, use
the dynamic client (`k8s.io/client-go/dynamic`) to fetch resources by GVR
(GroupVersionResource). This lets you handle Services, Deployments, StatefulSets,
and custom resources with the same code path — just parse `spec.selector` or
`spec.selector.matchLabels` from the unstructured object.

## Reconnection

SPDY tunnels drop. Pods get evicted. Nodes go away. If you're building anything
beyond a one-shot debug tool, you need reconnection logic.

### When to Reconnect

Not all forwards should reconnect:

- **Selector-based targets** (Services, Deployments): yes. The selector can
  resolve to a new pod. The user's intent is "forward to this service," not
  "forward to this specific pod."
- **Direct pod targets**: no. The pod is gone; there's nothing to reconnect to.
  Clean up and notify the caller.

### Exponential Backoff

A tight retry loop hammering the API server during an outage is a bad neighbor.
Use exponential backoff with a cap:

```go
backoff := time.Second
maxBackoff := 30 * time.Second

for {
    err := runTunnel(ctx, ...)
    if ctx.Err() != nil {
        return // stopped intentionally
    }

    select {
    case <-ctx.Done():
        return
    case <-time.After(backoff):
        backoff = min(backoff*2, maxBackoff)
    }
}
```

**Reset backoff on success.** When a tunnel connects and runs for a while before
dropping, reset to the initial backoff. Otherwise a forward that's been stable
for hours will wait 30 seconds after a brief blip.

### Re-Resolving the Pod

On each reconnect iteration, re-resolve the pod from the selector. Don't cache
the pod name — it may have been replaced. The sequence is:

1. Resolve pod from selector.
2. Open tunnel.
3. Tunnel drops.
4. Go to 1 (not 2 — the old pod may be gone).

## Gotchas and Hard-Won Lessons

### `ForwardPorts()` Blocks — Plan Your Goroutine Model

`ForwardPorts()` doesn't return until the tunnel dies. Each active forward needs
its own goroutine. If you're managing multiple forwards, you need a registry
(map of ID to cancel function) and a mutex.

### Port Conflicts

If you specify a local port that's already in use, `ForwardPorts()` will fail
immediately. Either use `0` for auto-assign, or be prepared to handle
`bind: address already in use` errors gracefully.

### The Tunnel Can Die Silently

`ForwardPorts()` sometimes returns `nil` error even when the tunnel is gone (e.g.,
the pod was deleted). Don't assume a nil error means "stopped cleanly by the user."
Check your context to distinguish intentional stops from unexpected drops.

### SPDY Deprecation (Sort Of)

SPDY has been "deprecated" in Kubernetes in favor of WebSocket-based tunneling
(KEP-4006), but `client-go`'s portforwarder still uses SPDY as of v0.31. The API
server supports both. If you're targeting very new clusters, keep an eye on this —
but for now SPDY works everywhere.

### Auth and TLS

The SPDY transport inherits auth from `*rest.Config`. If you're using token-based
auth (e.g., exec-based credential plugins like `aws-iam-authenticator`), be aware
that token refresh can race with long-lived tunnels. The tunnel itself doesn't
re-authenticate — if the API server drops the connection due to an expired token,
you'll see an opaque error from the SPDY layer. Your reconnect loop handles this
naturally, but it's worth knowing why a tunnel that ran for exactly 1 hour
suddenly died.

### Testing Without a Cluster

The tunnel setup is tightly coupled to real HTTP connections. To unit test
reconnection and lifecycle logic, inject the tunnel as a function:

```go
type tunnelFunc func(ctx context.Context, ..., onReady func(uint16)) error
```

Tests can provide a fake that simulates success, failure, or delayed readiness
without any network I/O. Test the SPDY layer separately with integration tests
against a real cluster (or `envtest`).

## Summary

| Concern | Approach |
|---|---|
| Protocol | SPDY upgrade on pod `portforward` subresource |
| Library | `k8s.io/client-go/tools/portforward` + `transport/spdy` |
| Local port | Pass `0` to auto-assign, read from `GetPorts()` after ready |
| Pod resolution | Extract selector from parent resource, rank matching pods |
| Reconnection | Exponential backoff, re-resolve pod each iteration, only for selector targets |
| Lifecycle | One goroutine per tunnel, `stopCh`/context bridge, registry with mutex |
| Testing | Inject the tunnel function, test lifecycle logic in isolation |
