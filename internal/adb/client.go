package adb

import (
	"bytes"
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client manages a direct TCP ADB connection to a Fire TV Stick (:5555).
type Client struct {
	address       string
	conn          net.Conn
	privKey       *rsa.PrivateKey
	banner        string
	maxPayload    uint32
	nextLocalID   atomic.Uint32
	mu            sync.Mutex
	PromptHandler func() // Optional callback invoked when waiting for user authorization on the TV screen
}

// NormalizeAddress appends the default ADB port (:5555) if no port was provided.
func NormalizeAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, fmt.Sprintf("%d", DefaultPort))
}

// NewClient initializes a new pure-Go ADB Client with the given target IP/address and RSA key.
// If privKey is nil, it automatically loads or generates the default local ADB RSA key.
func NewClient(address string, privKey *rsa.PrivateKey) (*Client, error) {
	norm := NormalizeAddress(address)
	if norm == "" {
		return nil, fmt.Errorf("adb target address cannot be empty")
	}

	if privKey == nil {
		var err error
		privKey, err = LoadOrGenerateKey("")
		if err != nil {
			return nil, err
		}
	}

	c := &Client{
		address:    norm,
		privKey:    privKey,
		maxPayload: MaxPayload,
	}
	c.nextLocalID.Store(1)
	return c, nil
}

// Address returns the normalized host:port target address.
func (c *Client) Address() string {
	return c.address
}

// DeviceBanner returns the banner reported by the Fire Stick during the CNXN handshake.
func (c *Client) DeviceBanner() string {
	return c.banner
}

// Connect establishes the TCP connection and completes the ADB CNXN/AUTH handshake.
// If the Fire Stick does not recognize our RSA key yet, it sends the public key and waits
// until the user accepts the "Allow USB Debugging" prompt on the TV (or ctx expires).
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return fmt.Errorf("failed to connect to Fire Stick at %s: %w", c.address, err)
	}

	// Ensure deadline respects context cancellation during handshake
	stopCancel := context.AfterFunc(ctx, func() {
		_ = conn.SetDeadline(time.Now())
	})
	defer stopCancel()

	// Send initial CNXN packet
	hostBanner := []byte("host::\x00")
	if err := WritePacket(conn, CmdCNXN, ADBVersion, MaxPayload, hostBanner); err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to send ADB CNXN packet: %w", err)
	}

	pkt, err := ReadPacket(conn)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to read ADB handshake response: %w", err)
	}

	// Handle AUTH challenge if required by adbd
	if pkt.Command == CmdAUTH {
		if pkt.Arg0 != AuthToken {
			_ = conn.Close()
			return fmt.Errorf("unexpected ADB AUTH arg0: %d", pkt.Arg0)
		}

		// Step 1: Sign the 20-byte token with our RSA private key
		sig, err := SignToken(c.privKey, pkt.Data)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to sign ADB token: %w", err)
		}

		if err := WritePacket(conn, CmdAUTH, AuthSignature, 0, sig); err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to send ADB AUTH signature: %w", err)
		}

		pkt, err = ReadPacket(conn)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to read ADB response after signature: %w", err)
		}

		// Step 2: If device still returns AUTH, our key is not yet trusted -> send RSAPublicKey
		if pkt.Command == CmdAUTH {
			pubPayload, err := EncodeAndroidPublicKey(&c.privKey.PublicKey, "firestick_debloat@local")
			if err != nil {
				_ = conn.Close()
				return fmt.Errorf("failed to encode ADB public key: %w", err)
			}

			if c.PromptHandler != nil {
				c.PromptHandler()
			}

			if err := WritePacket(conn, CmdAUTH, AuthRSAPublicKey, 0, pubPayload); err != nil {
				_ = conn.Close()
				return fmt.Errorf("failed to send ADB RSA public key: %w", err)
			}

			// Wait for the user to press "OK" on the Fire Stick TV prompt
			pkt, err = ReadPacket(conn)
			if err != nil {
				_ = conn.Close()
				return fmt.Errorf("waiting for TV authorization failed (did you allow debugging on the TV?): %w", err)
			}
		}
	}

	if pkt.Command != CmdCNXN {
		_ = conn.Close()
		return fmt.Errorf("expected ADB CNXN, got %s (0x%x)", CommandName(pkt.Command), pkt.Command)
	}

	if pkt.Arg1 > 0 {
		c.maxPayload = pkt.Arg1
	}
	c.banner = string(bytes.TrimRight(pkt.Data, "\x00"))
	c.conn = conn
	_ = c.conn.SetDeadline(time.Time{})
	return nil
}

// RunShellCommand opens a "shell:<cmd>" stream on the Fire Stick, collects all output,
// and closes the stream cleanly.
func (c *Client) RunShellCommand(ctx context.Context, cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return "", fmt.Errorf("adb client is not connected")
	}

	stopCancel := context.AfterFunc(ctx, func() {
		_ = c.conn.SetDeadline(time.Now())
	})
	defer func() {
		stopCancel()
		if c.conn != nil {
			_ = c.conn.SetDeadline(time.Time{})
		}
	}()

	localID := c.nextLocalID.Add(1)
	openPayload := append([]byte("shell:"+cmd), 0x00)

	if err := WritePacket(c.conn, CmdOPEN, localID, 0, openPayload); err != nil {
		return "", fmt.Errorf("failed to open shell stream for %q: %w", cmd, err)
	}

	var (
		remoteID uint32
		output   bytes.Buffer
	)

	for {
		pkt, err := ReadPacket(c.conn)
		if err != nil {
			return output.String(), fmt.Errorf("error reading shell stream for %q: %w", cmd, err)
		}

		switch pkt.Command {
		case CmdOKAY:
			if remoteID == 0 {
				remoteID = pkt.Arg0
			}
		case CmdWRTE:
			if remoteID == 0 {
				remoteID = pkt.Arg0
			}
			output.Write(pkt.Data)
			// Acknowledge receipt so adbd sends the next chunk or CLSE
			if err := WritePacket(c.conn, CmdOKAY, localID, remoteID, nil); err != nil {
				return output.String(), fmt.Errorf("failed to send OKAY ack: %w", err)
			}
		case CmdCLSE:
			// Acknowledge stream closure
			_ = WritePacket(c.conn, CmdCLSE, localID, remoteID, nil)
			return strings.TrimSpace(output.String()), nil
		default:
			return output.String(), fmt.Errorf("unexpected ADB command %s during shell execution", CommandName(pkt.Command))
		}
	}
}

// Close closes the underlying TCP connection to the Fire Stick.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
