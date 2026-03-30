package portforward

import (
	"fmt"
	"io"
	"net/http"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/Vilsol/klados/internal/cluster"
	"golang.org/x/net/context"
)

// defaultRunTunnel establishes a SPDY port-forward tunnel to podName.
// It calls onReady with the assigned local port once the tunnel is established,
// then blocks until the tunnel drops or ctx is cancelled.
func defaultRunTunnel(ctx context.Context, conn *cluster.Connection, namespace, podName string, localPort, remotePort int, onReady func(uint16)) error {
	url := conn.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(conn.Config)
	if err != nil {
		return fmt.Errorf("creating SPDY transport: %w", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})

	go func() {
		<-ctx.Done()
		close(stopCh)
	}()

	fw, err := portforward.New(dialer, ports, stopCh, readyCh, io.Discard, io.Discard)
	if err != nil {
		return fmt.Errorf("creating port-forwarder: %w", err)
	}

	go func() {
		<-readyCh
		if onReady != nil {
			fwPorts, _ := fw.GetPorts()
			if len(fwPorts) > 0 {
				onReady(fwPorts[0].Local)
			}
		}
	}()

	return fw.ForwardPorts()
}
