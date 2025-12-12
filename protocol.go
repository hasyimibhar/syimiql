package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"crypto/rand"
)

const (
	protocolVersion3_0 = 196608
	sslRequestCode     = 80877103
)

type PgServer struct {
	listener net.Listener
}

var connectionCounter atomic.Uint32

func NewPgServer(addr string) (*PgServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &PgServer{listener: listener}, nil
}

func (s *PgServer) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *PgServer) Close() error {
	return s.listener.Close()
}

func (s *PgServer) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(conn)
	}
}

func generateSecret() uint32 {
    var secret uint32
    binary.Read(rand.Reader, binary.BigEndian, &secret)
    return secret
}

func (s *PgServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	pid := connectionCounter.Add(1)
	secret := generateSecret()

	if err := s.handleStartup(conn, pid, secret); err != nil {
		s.sendError(conn, err.Error())
		return
	}

	for {
		msgType, payload, err := s.readMessage(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}

		switch msgType {
		case 'Q':
			s.handleQuery(conn, payload)
		case 'X':
			return
		default:
			s.sendError(conn, fmt.Sprintf("unsupported message type: %c", msgType))
			return
		}
	}
}

func (s *PgServer) handleStartup(conn net.Conn, pid, secret uint32) error {
	for {
		lengthBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lengthBuf); err != nil {
			return err
		}
		length := int(binary.BigEndian.Uint32(lengthBuf))

		payload := make([]byte, length-4)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return err
		}

		version := binary.BigEndian.Uint32(payload[:4])

		if version == sslRequestCode {
			if _, err := conn.Write([]byte{'N'}); err != nil {
				return err
			}
			continue
		}

		if version != protocolVersion3_0 {
			return fmt.Errorf("unsupported protocol version: %d", version)
		}

		params := s.parseStartupParams(payload[4:])
		_ = params
		break
	}

	if err := s.sendAuthenticationOk(conn); err != nil {
		return err
	}

	if err := s.sendParameterStatus(conn, "server_version", "15.0"); err != nil {
		return err
	}
	if err := s.sendParameterStatus(conn, "client_encoding", "UTF8"); err != nil {
		return err
	}
	if err := s.sendParameterStatus(conn, "server_encoding", "UTF8"); err != nil {
		return err
	}

	if err := s.sendBackendKeyData(conn, pid, secret); err != nil {
		return err
	}

	if err := s.sendReadyForQuery(conn, 'I'); err != nil {
		return err
	}

	return nil
}

func (s *PgServer) parseStartupParams(data []byte) map[string]string {
	params := make(map[string]string)
	i := 0
	for i < len(data) {
		keyEnd := i
		for keyEnd < len(data) && data[keyEnd] != 0 {
			keyEnd++
		}
		if keyEnd >= len(data) {
			break
		}
		key := string(data[i:keyEnd])
		if key == "" {
			break
		}
		i = keyEnd + 1

		valEnd := i
		for valEnd < len(data) && data[valEnd] != 0 {
			valEnd++
		}
		value := string(data[i:valEnd])
		i = valEnd + 1

		params[key] = value
	}
	return params
}

func (s *PgServer) readMessage(conn net.Conn) (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	msgType := header[0]
	length := int(binary.BigEndian.Uint32(header[1:5]))

	payload := make([]byte, length-4)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, nil, err
	}

	return msgType, payload, nil
}

func (s *PgServer) handleQuery(conn net.Conn, payload []byte) {
	query := string(payload[:len(payload)-1])

	stmt, err := parseStmt(query)
	if err != nil {
		s.sendError(conn, "syntax error")
		s.sendReadyForQuery(conn, 'I')
		return
	}

	_ = stmt

	// Test
    if _, ok := stmt.(*SelectStmt); ok {
        fields := []FieldDescription{
            {
                Name:       "?column?",  // or the actual column name/alias
                TypeOID:    23,          // 23 = int4, 25 = text, 701 = float8
                TypeSize:   4,           // size in bytes (-1 for variable)
                TypeMod:    -1,
                FormatCode: 0,           // 0 = text format
            },
        }
        s.sendRowDescription(conn, fields)

        s.sendDataRow(conn, []string{"100"})

        s.sendCommandComplete(conn, "SELECT 1")
        s.sendReadyForQuery(conn, 'I')
        return
    }

	s.sendCommandComplete(conn, "OK")
	s.sendReadyForQuery(conn, 'I')
}

func (s *PgServer) sendAuthenticationOk(conn net.Conn) error {
	msg := make([]byte, 9)
	msg[0] = 'R'
	binary.BigEndian.PutUint32(msg[1:5], 8)
	binary.BigEndian.PutUint32(msg[5:9], 0)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendParameterStatus(conn net.Conn, name, value string) error {
	payload := append([]byte(name), 0)
	payload = append(payload, []byte(value)...)
	payload = append(payload, 0)

	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'S'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendBackendKeyData(conn net.Conn, pid, secret uint32) error {
	msg := make([]byte, 13)
	msg[0] = 'K'
	binary.BigEndian.PutUint32(msg[1:5], 12)
	binary.BigEndian.PutUint32(msg[5:9], pid)
	binary.BigEndian.PutUint32(msg[9:13], secret)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendReadyForQuery(conn net.Conn, status byte) error {
	msg := []byte{'Z', 0, 0, 0, 5, status}
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendCommandComplete(conn net.Conn, tag string) error {
	payload := append([]byte(tag), 0)
	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'C'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}

type FieldDescription struct {
	Name         string
	TableOID     int32
	ColumnAttrNo int16
	TypeOID      int32
	TypeSize     int16
	TypeMod      int32
	FormatCode   int16
}

func (s *PgServer) sendRowDescription(conn net.Conn, fields []FieldDescription) error {
	var payload []byte

	// Number of fields
	fieldCount := make([]byte, 2)
	binary.BigEndian.PutUint16(fieldCount, uint16(len(fields)))
	payload = append(payload, fieldCount...)

	// Each field
	for _, field := range fields {
		// Field name (null-terminated string)
		payload = append(payload, []byte(field.Name)...)
		payload = append(payload, 0)

		// Table OID (4 bytes)
		tableOID := make([]byte, 4)
		binary.BigEndian.PutUint32(tableOID, uint32(field.TableOID))
		payload = append(payload, tableOID...)

		// Column attribute number (2 bytes)
		attrNo := make([]byte, 2)
		binary.BigEndian.PutUint16(attrNo, uint16(field.ColumnAttrNo))
		payload = append(payload, attrNo...)

		// Type OID (4 bytes)
		typeOID := make([]byte, 4)
		binary.BigEndian.PutUint32(typeOID, uint32(field.TypeOID))
		payload = append(payload, typeOID...)

		// Type size (2 bytes)
		typeSize := make([]byte, 2)
		binary.BigEndian.PutUint16(typeSize, uint16(field.TypeSize))
		payload = append(payload, typeSize...)

		// Type modifier (4 bytes)
		typeMod := make([]byte, 4)
		binary.BigEndian.PutUint32(typeMod, uint32(field.TypeMod))
		payload = append(payload, typeMod...)

		// Format code (2 bytes) - 0 for text, 1 for binary
		formatCode := make([]byte, 2)
		binary.BigEndian.PutUint16(formatCode, uint16(field.FormatCode))
		payload = append(payload, formatCode...)
	}

	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'T'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendDataRow(conn net.Conn, values []string) error {
	var payload []byte

	// Number of columns
	colCount := make([]byte, 2)
	binary.BigEndian.PutUint16(colCount, uint16(len(values)))
	payload = append(payload, colCount...)

	// Each column value
	for _, value := range values {
		// Length of value (4 bytes) - -1 for NULL
		length := make([]byte, 4)
		binary.BigEndian.PutUint32(length, uint32(len(value)))
		payload = append(payload, length...)

		// Value (no null terminator for DataRow)
		payload = append(payload, []byte(value)...)
	}

	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'D'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}

type ErrorDetail struct {
	Severity string
	Code     string
	Message  string
	Detail   string
	Hint     string
	Position int
}

func (s *PgServer) sendError(conn net.Conn, message string) error {
	return s.sendErrorDetail(conn, ErrorDetail{
		Severity: "ERROR",
		Code:     "42000", // syntax_error_or_access_rule_violation
		Message:  message,
	})
}

func (s *PgServer) sendErrorDetail(conn net.Conn, detail ErrorDetail) error {
	var payload []byte

	// Severity
	payload = append(payload, 'S')
	payload = append(payload, []byte(detail.Severity)...)
	payload = append(payload, 0)

	// SQLSTATE code
	payload = append(payload, 'C')
	payload = append(payload, []byte(detail.Code)...)
	payload = append(payload, 0)

	// Message
	payload = append(payload, 'M')
	payload = append(payload, []byte(detail.Message)...)
	payload = append(payload, 0)

	// Detail (optional)
	if detail.Detail != "" {
		payload = append(payload, 'D')
		payload = append(payload, []byte(detail.Detail)...)
		payload = append(payload, 0)
	}

	// Hint (optional)
	if detail.Hint != "" {
		payload = append(payload, 'H')
		payload = append(payload, []byte(detail.Hint)...)
		payload = append(payload, 0)
	}

	// Position (optional)
	if detail.Position > 0 {
		payload = append(payload, 'P')
		payload = append(payload, []byte(fmt.Sprintf("%d", detail.Position))...)
		payload = append(payload, 0)
	}

	// Terminator
	payload = append(payload, 0)

	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'E'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}
