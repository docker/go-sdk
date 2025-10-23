package container

import (
	"context"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-connections/nat"
)

// TestUDPPortBinding tests the fix for the UDP port binding issue.
// This addresses the bug where exposed UDP ports always returned "0" instead of the actual mapped port.
//
// Background: When using ExposedPorts: []string{"8080/udp"}, the MappedPort() function
// would return "0/udp" instead of the actual host port like "55051/udp".
//
// Root cause: nat.ParsePortSpecs() creates PortBinding with empty HostPort (""),
// but Docker needs HostPort: "0" for automatic port allocation.
//
// Fix: In mergePortBindings(), convert empty HostPort to "0" for auto-allocation.
func TestUDPPortBinding(t *testing.T) {
	ctx := context.Background()

	t.Run("UDP port gets proper host port allocation", func(t *testing.T) {
		// Create container with UDP port exposed
		c, err := Run(ctx,
			WithImage("alpine/socat:latest"),
			WithExposedPorts("8080/udp"),
			WithCmd("UDP-LISTEN:8080,fork,reuseaddr", "EXEC:'/bin/cat'"),
		)
		require.NoError(t, err)
		Cleanup(t, c)

		// Test MappedPort function - this was the bug
		udpPort, ok := network.PortFrom(8080, "udp")
		require.True(t, ok)

		mappedPort, err := c.MappedPort(ctx, udpPort)
		require.NoError(t, err)

		// Before fix: mappedPort.Port() would return "0"
		// After fix: mappedPort.Port() returns actual port like "55051"
		require.NotEqual(t, 0, mappedPort.Num(), "UDP port should not return '0'")
		require.Equal(t, network.UDP, mappedPort.Proto(), "Protocol should be UDP")

		// Verify the port is actually accessible (basic connectivity test)
		hostIP, err := c.Host(ctx)
		require.NoError(t, err)

		address := net.JoinHostPort(hostIP, strconv.Itoa(int(mappedPort.Num())))
		conn, err := net.DialTimeout("udp", address, 2*time.Second)
		require.NoError(t, err, "Should be able to connect to UDP port")
		conn.Close()
	})

	t.Run("TCP port continues to work (regression test)", func(t *testing.T) {
		// Ensure our UDP fix doesn't break TCP ports
		c, err := Run(ctx,
			WithImage("nginx:alpine"),
			WithExposedPorts("80/tcp"),
		)
		require.NoError(t, err)
		Cleanup(t, c)

		tcpPort, ok := network.PortFrom(80, "tcp")
		require.True(t, ok)

		mappedPort, err := c.MappedPort(ctx, tcpPort)
		require.NoError(t, err)

		assert.NotEqual(t, 0, mappedPort.Num(), "TCP port should not return '0'")
		assert.Equal(t, network.TCP, mappedPort.Proto(), "Protocol should be TCP")
	})
}

// TestPortBindingInternalLogic tests the internal mergePortBindings function
// that was modified to fix the UDP port binding issue.
func TestPortBindingInternalLogic(t *testing.T) {
	t.Run("mergePortBindings fixes empty HostPort", func(t *testing.T) {
		// Test the core fix: empty HostPort should become "0"
		// This simulates what nat.ParsePortSpecs returns for "8080/udp"
		exposedPortMap := network.PortMap{
			network.MustParsePort("8080/udp"): []network.PortBinding{{}}, // Empty HostPort (the bug)
		}
		configPortMap := network.PortMap{} // No existing port bindings
		exposedPorts := []string{"8080/udp"}

		// Call the function our fix modified
		result := mergePortBindings(configPortMap, exposedPortMap, exposedPorts)

		// Verify the fix worked
		udp80 := network.MustParsePort("8080/udp")
		require.Contains(t, result, udp80)
		bindings := result[udp80]
		require.Len(t, bindings, 1)

		// THE KEY ASSERTION: Empty HostPort should become "0"
		assert.Equal(t, "0", bindings[0].HostPort,
			"Empty HostPort should be converted to '0' for auto-allocation")
		assert.Empty(t, bindings[0].HostIP, "HostIP should remain empty for all interfaces")
	})

	t.Run("mergePortBindings preserves existing HostPort", func(t *testing.T) {
		// Ensure we don't modify already-set HostPort values
		exposedPortMap := network.PortMap{
			network.MustParsePort("8080/udp"): []network.PortBinding{{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "9090"}},
		}
		configPortMap := network.PortMap{}
		exposedPorts := []string{"8080/udp"}

		result := mergePortBindings(configPortMap, exposedPortMap, exposedPorts)

		bindings := result[network.MustParsePort("8080/udp")]
		require.Len(t, bindings, 1)

		// Should preserve existing values
		assert.Equal(t, "9090", bindings[0].HostPort, "Existing HostPort should be preserved")
		assert.Equal(t, "127.0.0.1", bindings[0].HostIP.String(), "Existing HostIP should be preserved")
	})

	t.Run("nat.ParsePortSpecs behavior documentation", func(t *testing.T) {
		// This test documents the behavior of nat.ParsePortSpecs that caused the bug
		exposedPorts := []string{"8080/udp", "9090/tcp"}
		exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs(exposedPorts)
		require.NoError(t, err)

		// Verify the port set
		assert.Contains(t, exposedPortSet, nat.Port("8080/udp"))
		assert.Contains(t, exposedPortSet, nat.Port("9090/tcp"))

		// Document the problematic behavior: nat.ParsePortSpecs creates empty HostPort
		udpBindings := exposedPortMap["8080/udp"]
		require.Len(t, udpBindings, 1)
		assert.Empty(t, udpBindings[0].HostPort,
			"nat.ParsePortSpecs creates empty HostPort (this was the source of the bug)")

		tcpBindings := exposedPortMap["9090/tcp"]
		require.Len(t, tcpBindings, 1)
		assert.Empty(t, tcpBindings[0].HostPort,
			"nat.ParsePortSpecs creates empty HostPort for all protocols")
	})
}
