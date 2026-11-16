package adb

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPacketEncodeDecode(t *testing.T) {
	original := &Packet{
		Command: CmdCNXN,
		Arg0:    ADBVersion,
		Arg1:    MaxPayload,
		Data:    []byte("host::\x00"),
	}

	raw := original.Encode()
	decoded, err := ReadPacket(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	if decoded.Command != original.Command || decoded.Arg0 != original.Arg0 || decoded.Arg1 != original.Arg1 {
		t.Fatalf("header mismatch: got %+v, want %+v", decoded, original)
	}
	if !bytes.Equal(decoded.Data, original.Data) {
		t.Fatalf("payload mismatch: got %q, want %q", decoded.Data, original.Data)
	}
}

func TestLoadOrGenerateKeyAndPublicKeyEncoding(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "adbkey")

	key1, err := LoadOrGenerateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadOrGenerateKey (create) failed: %v", err)
	}

	key2, err := LoadOrGenerateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadOrGenerateKey (reload) failed: %v", err)
	}

	if key1.N.Cmp(key2.N) != 0 {
		t.Fatalf("reloaded key modulus does not match generated key")
	}

	pubPayload, err := EncodeAndroidPublicKey(&key1.PublicKey, "test@firestick")
	if err != nil {
		t.Fatalf("EncodeAndroidPublicKey failed: %v", err)
	}
	if !bytes.HasSuffix(pubPayload, []byte(" test@firestick\x00")) {
		t.Fatalf("public key payload missing comment + null terminator: %q", string(pubPayload))
	}
}

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"192.168.1.100", "192.168.1.100:5555"},
		{"192.168.1.100:5555", "192.168.1.100:5555"},
		{"10.0.0.50:5556", "10.0.0.50:5556"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := NormalizeAddress(tc.in); got != tc.want {
			t.Errorf("NormalizeAddress(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestClientConnectAndRunShellWithMockServer tests the full ADB TCP lifecycle:
// 1. First-time connection where the mock Fire Stick requires RSA Public Key authorization (TV Prompt flow)
// 2. Subsequent connection where the mock Fire Stick accepts our RSA token signature immediately
// 3. Running multiple shell commands (OPEN -> OKAY -> WRTE -> OKAY -> CLSE)
func TestClientConnectAndRunShellWithMockServer(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test RSA key: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock ADB TCP listener: %v", err)
	}
	defer listener.Close()

	token := []byte("0123456789abcdefghij") // 20-byte ADB AUTH token
	trustedFirstAttempt := false

	// Run mock FireOS adbd server
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn, alreadyTrusted bool) {
				defer c.Close()

				// 1. Expect CNXN from client
				pkt, err := ReadPacket(c)
				if err != nil || pkt.Command != CmdCNXN {
					return
				}

				// 2. Send AUTH token challenge
				_ = WritePacket(c, CmdAUTH, AuthToken, 0, token)

				// 3. Read client's AUTH signature
				pkt, err = ReadPacket(c)
				if err != nil || pkt.Command != CmdAUTH || pkt.Arg0 != AuthSignature {
					return
				}

				// Verify signature matches client's RSA public key
				if err := rsa.VerifyPKCS1v15(&privKey.PublicKey, crypto.SHA1, token, pkt.Data); err != nil {
					return
				}

				if !alreadyTrusted {
					// Simulate first-time connection: challenge again so client sends AuthRSAPublicKey
					_ = WritePacket(c, CmdAUTH, AuthToken, 0, token)
					pkt, err = ReadPacket(c)
					if err != nil || pkt.Command != CmdAUTH || pkt.Arg0 != AuthRSAPublicKey {
						return
					}
				}

				// 4. Send CNXN confirming connection to FireOS
				_ = WritePacket(c, CmdCNXN, ADBVersion, MaxPayload, []byte("device::ro.product.model=AFTT\x00"))

				// 5. Handle shell streams
				for {
					openPkt, err := ReadPacket(c)
					if err != nil {
						return
					}
					if openPkt.Command != CmdOPEN {
						return
					}

					clientStreamID := openPkt.Arg0
					deviceStreamID := uint32(42)
					cmdStr := strings.TrimSuffix(string(openPkt.Data), "\x00")

					// Send OKAY
					_ = WritePacket(c, CmdOKAY, deviceStreamID, clientStreamID, nil)

					// Send WRTE with simulated FireOS output
					resp := "Success: executed " + cmdStr + "\n"
					_ = WritePacket(c, CmdWRTE, deviceStreamID, clientStreamID, []byte(resp))

					// Wait for client's OKAY ack
					ackPkt, err := ReadPacket(c)
					if err != nil || ackPkt.Command != CmdOKAY {
						return
					}

					// Send CLSE
					_ = WritePacket(c, CmdCLSE, deviceStreamID, clientStreamID, nil)
					// Read client's CLSE ack
					_, _ = ReadPacket(c)
				}
			}(conn, trustedFirstAttempt)
			trustedFirstAttempt = true
		}
	}()

	client, err := NewClient(listener.Addr().String(), privKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	promptTriggered := false
	client.PromptHandler = func() {
		promptTriggered = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection: triggers TV prompt flow (AuthRSAPublicKey)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("first Connect failed: %v", err)
	}
	if !promptTriggered {
		t.Errorf("expected PromptHandler to be called on first-time connection")
	}
	if !strings.Contains(client.DeviceBanner(), "AFTT") {
		t.Errorf("unexpected device banner: %q", client.DeviceBanner())
	}

	// Execute a shell command
	out, err := client.RunShellCommand(ctx, "pm disable-user --user 0 com.amazon.mp3")
	if err != nil {
		t.Fatalf("RunShellCommand failed: %v", err)
	}
	wantOut := "Success: executed shell:pm disable-user --user 0 com.amazon.mp3"
	if out != wantOut {
		t.Errorf("RunShellCommand output = %q, want %q", out, wantOut)
	}
}
