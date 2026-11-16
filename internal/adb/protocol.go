package adb

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ADB Protocol Commands (Little-Endian uint32 representations of 4-char ASCII codes)
const (
	CmdCNXN uint32 = 0x4e584e43 // "CNXN" - Connect
	CmdAUTH uint32 = 0x48545541 // "AUTH" - Authentication
	CmdOPEN uint32 = 0x4e45504f // "OPEN" - Open stream
	CmdOKAY uint32 = 0x59414b4f // "OKAY" - Ready / Acknowledge
	CmdCLSE uint32 = 0x45534c43 // "CLSE" - Close stream
	CmdWRTE uint32 = 0x45545257 // "WRTE" - Write data
)

// ADB Protocol Constants
const (
	ADBVersion       uint32 = 0x01000000 // ADB protocol version
	MaxPayload       uint32 = 256 * 1024 // 256 KB max payload
	DefaultPort      int    = 5555       // Standard TCP ADB port on FireOS
	HeaderLength     int    = 24         // 6 * uint32
	AuthToken        uint32 = 1          // ADB_AUTH_TOKEN
	AuthSignature    uint32 = 2          // ADB_AUTH_SIGNATURE
	AuthRSAPublicKey uint32 = 3          // ADB_AUTH_RSAPUBLICKEY
)

// Packet represents an ADB transport layer packet (24-byte header + variable data payload).
type Packet struct {
	Command uint32
	Arg0    uint32
	Arg1    uint32
	Data    []byte
}

// Checksum computes the ADB payload checksum (sum of all bytes in Data).
// In AOSP transport.cpp, data_check is a simple byte sum of the payload.
func Checksum(data []byte) uint32 {
	var sum uint32
	for _, b := range data {
		sum += uint32(b)
	}
	return sum
}

// Magic returns the inverted command value required in the ADB header.
func Magic(cmd uint32) uint32 {
	return cmd ^ 0xffffffff
}

// Encode serializes the Packet into its wire format (24-byte Little-Endian header + payload).
func (p *Packet) Encode() []byte {
	buf := make([]byte, HeaderLength+len(p.Data))
	binary.LittleEndian.PutUint32(buf[0:4], p.Command)
	binary.LittleEndian.PutUint32(buf[4:8], p.Arg0)
	binary.LittleEndian.PutUint32(buf[8:12], p.Arg1)
	binary.LittleEndian.PutUint32(buf[12:16], uint32(len(p.Data)))
	binary.LittleEndian.PutUint32(buf[16:20], Checksum(p.Data))
	binary.LittleEndian.PutUint32(buf[20:24], Magic(p.Command))
	copy(buf[HeaderLength:], p.Data)
	return buf
}

// WritePacket encodes and writes a packet to an io.Writer.
func WritePacket(w io.Writer, cmd, arg0, arg1 uint32, data []byte) error {
	pkt := Packet{
		Command: cmd,
		Arg0:    arg0,
		Arg1:    arg1,
		Data:    data,
	}
	raw := pkt.Encode()
	_, err := w.Write(raw)
	return err
}

// ReadPacket reads and validates a single ADB Packet from an io.Reader.
func ReadPacket(r io.Reader) (*Packet, error) {
	hdr := make([]byte, HeaderLength)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}

	cmd := binary.LittleEndian.Uint32(hdr[0:4])
	arg0 := binary.LittleEndian.Uint32(hdr[4:8])
	arg1 := binary.LittleEndian.Uint32(hdr[8:12])
	dataLen := binary.LittleEndian.Uint32(hdr[12:16])
	dataCheck := binary.LittleEndian.Uint32(hdr[16:20])
	magic := binary.LittleEndian.Uint32(hdr[20:24])

	if magic != Magic(cmd) {
		return nil, fmt.Errorf("adb protocol error: invalid magic 0x%x for command 0x%x", magic, cmd)
	}

	if dataLen > MaxPayload*4 {
		return nil, fmt.Errorf("adb protocol error: payload length %d exceeds safety limit", dataLen)
	}

	var data []byte
	if dataLen > 0 {
		data = make([]byte, dataLen)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		if actual := Checksum(data); actual != dataCheck {
			return nil, fmt.Errorf("adb protocol error: checksum mismatch (expected 0x%x, got 0x%x)", dataCheck, actual)
		}
	}

	return &Packet{
		Command: cmd,
		Arg0:    arg0,
		Arg1:    arg1,
		Data:    data,
	}, nil
}

// CommandName returns the 4-character ASCII name for an ADB command.
func CommandName(cmd uint32) string {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], cmd)
	return string(b[:])
}
